package letsencrypt

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/go-acme/lego/v3/certcrypto"
	"github.com/go-acme/lego/v3/certificate"
	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/challenge/dns01"
	lego "github.com/go-acme/lego/v3/lego"
	loge "github.com/go-acme/lego/v3/log"
	"github.com/go-acme/lego/v3/registration"
)

const (
	StorageDir       = "/etc/letsencrypt"
	ProductionApiUri = "https://acme-v02.api.letsencrypt.org/directory"
	StagingApiUri    = "https://acme-staging-v02.api.letsencrypt.org/directory"
)

type KeyType string

const (
	RSA2048 KeyType = "RSA-2048"
	RSA4096 KeyType = "RSA-4096"
	RSA8192 KeyType = "RSA-8192"
	EC256   KeyType = "ECDSA-256"
	EC384   KeyType = "ECDSA-384"
)

type ApiVersion string

const (
	Production ApiVersion = "Production"
	Sandbox    ApiVersion = "Sandbox"
)

// AcmeCertificate represents a CA issued certificate,
// PrivateKey and Certificate are both PEM encoded.
//
// Anonymous fields:
// PrivateKey  []byte
// Certificate []byte
// Domain      string
type AcmeCertificate struct {
	certificate.Resource
	DnsNames     string    `json:"dnsNames"`
	ExpiryDate   time.Time `json:"expiryDate"`
	SerialNumber string    `json:"serialNumber"`
}

// Client represents a Lets Encrypt client
type Client struct {
	client     *lego.Client
	apiVersion ApiVersion
	provider   Provider
}

// NewClient returns a new Lets Encrypt client
func NewClient(email string, kt KeyType, apiVer ApiVersion, dnsResolvers []string, provider ProviderOpts) (*Client, error) {
	var keyType certcrypto.KeyType
	switch kt {
	case RSA2048:
		keyType = certcrypto.RSA2048
	case RSA4096:
		keyType = certcrypto.RSA4096
	case RSA8192:
		keyType = certcrypto.RSA8192
	case EC256:
		keyType = certcrypto.EC256
	case EC384:
		keyType = certcrypto.EC384
	default:
		return nil, fmt.Errorf("Invalid private key type: %s", string(kt))
	}

	var serverUri string
	switch apiVer {
	case Production:
		serverUri = lego.LEDirectoryProduction
	case Sandbox:
		serverUri = lego.LEDirectoryStaging
	default:
		return nil, fmt.Errorf("Invalid API version: %s", string(apiVer))
	}

	acc, err := NewAccount(email, apiVer, keyType)
	if err != nil {
		return nil, fmt.Errorf("Could not initialize account store for %s: %v", email, err)
	}

	config := lego.NewConfig(acc)
	config.CADirURL = serverUri
	config.Certificate.KeyType = keyType

	client, err := lego.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("Could not create client: %v", err)
	}

	loge.Logger = log.New(ioutil.Discard, "", 0)

	prov, challengeType, err := getProvider(provider)
	if err != nil {
		return nil, fmt.Errorf("Could not get provider: %v", err)
	}

	if challengeType == challenge.DNS01 {
		err = client.Challenge.SetDNS01Provider(prov)
		if err != nil {
			return nil, fmt.Errorf("Could not set provider: %v", err)
		}
	} else if challengeType == challenge.HTTP01 {
		err = client.Challenge.SetHTTP01Provider(prov)
		if err != nil {
			return nil, fmt.Errorf("Could not set provider: %v", err)
		}
	}

	if acc.Registration == nil {
		logrus.Infof("Creating Let's Encrypt account for %s", email)
		reg, err := client.Registration.Register(registration.RegisterOptions{TermsOfServiceAgreed: true})
		if err != nil {
			return nil, fmt.Errorf("Failed to register account: %v", err)
		}

		acc.Registration = reg
		err = acc.Save()
		if err != nil {
			logrus.Errorf("Could not save account data: %v", err)
		}
	} else {
		logrus.Infof("Using locally stored Let's Encrypt account for %s", email)
	}

	if len(dnsResolvers) > 0 {
		dns01.AddRecursiveNameservers(dnsResolvers)
	}

	return &Client{
		client:     client,
		apiVersion: apiVer,
		provider:   provider.Provider,
	}, nil
}

// EnableLogs prints logs from the upstream lego library
func (c *Client) EnableLogs() {
	logger := logrus.New()
	logger.Out = os.Stdout
	loge.Logger = log.New(logger.Writer(), "", 0)
}

// Issue obtains a new SAN certificate from the Lets Encrypt CA
func (c *Client) Issue(certName string, domains []string) (*AcmeCertificate, error) {
	obtainRequest := certificate.ObtainRequest{
		Domains:    domains,
		Bundle:     true,
		PrivateKey: nil,
		MustStaple: false,
	}

	certRes, err := c.client.Certificate.Obtain(obtainRequest)
	if err != nil {
		return nil, err
	}

	dnsNames := dnsNamesIdentifier(domains)
	acmeCert, err := c.saveCertificate(certName, dnsNames, certRes)
	if err != nil {
		logrus.Fatalf("Error saving certificate '%s': %v", certName, err)
		return nil, err
	}

	return acmeCert, nil
}

// Renew renewes the given stored certificate
func (c *Client) Renew(certName string) (*AcmeCertificate, error) {
	acmeCert, err := c.loadCertificateByName(certName)
	if err != nil {
		return nil, fmt.Errorf("Error loading certificate '%s': %v", certName, err)
	}

	certRes := acmeCert.Resource
	newCertRes, err := c.client.Certificate.Renew(certRes, true, false)
	if err != nil {
		return nil, err
	}

	newAcmeCert, err := c.saveCertificate(certName, acmeCert.DnsNames, newCertRes)
	if err != nil {
		logrus.Fatalf("Error saving certificate '%s': %v", certName, err)
	}

	return newAcmeCert, nil
}

// GetStoredCertificate returns the locally stored certificate for the given domains
func (c *Client) GetStoredCertificate(certName string, domains []string) (bool, *AcmeCertificate) {
	logrus.Debugf("Looking up stored certificate by name '%s'", certName)
	if !c.haveCertificateByName(certName) {
		return false, nil
	}

	acmeCert, err := c.loadCertificateByName(certName)
	if err != nil {
		// Don't quit. Try to issue a new certificate instead.
		logrus.Errorf("Error loading certificate '%s': %v", certName, err)
		return false, nil
	}

	// check if the DNS names are a match
	if dnsNames := dnsNamesIdentifier(domains); acmeCert.DnsNames != dnsNames {
		logrus.Infof("Stored certificate does not have matching domain names: '%s' ", acmeCert.DnsNames)
		return false, nil
	}

	return true, &acmeCert
}

func (c *Client) haveCertificateByName(certName string) bool {
	certPath := c.CertPath(certName)
	if _, err := os.Stat(path.Join(certPath, "metadata.json")); err != nil {
		logrus.Debugf("No certificate in path '%s'", certPath)
		return false
	}

	return true
}

func (c *Client) loadCertificateByName(certName string) (AcmeCertificate, error) {
	var acmeCert AcmeCertificate
	certPath := c.CertPath(certName)

	logrus.Debugf("Loading certificate '%s' from '%s'", certName, certPath)

	certIn := path.Join(certPath, "fullchain.pem")
	privIn := path.Join(certPath, "privkey.pem")
	metaIn := path.Join(certPath, "metadata.json")

	certBytes, err := ioutil.ReadFile(certIn)
	if err != nil {
		return acmeCert, fmt.Errorf("Failed to load certificate from '%s': %v", certIn, err)
	}

	metaBytes, err := ioutil.ReadFile(metaIn)
	if err != nil {
		return acmeCert, fmt.Errorf("Failed to load meta data from '%s': %v", metaIn, err)
	}

	keyBytes, err := ioutil.ReadFile(privIn)
	if err != nil {
		return acmeCert, fmt.Errorf("Failed to load private key from '%s': %v", privIn, err)
	}

	err = json.Unmarshal(metaBytes, &acmeCert)
	if err != nil {
		return acmeCert, fmt.Errorf("Failed to unmarshal json meta data from '%s': %v", metaIn, err)
	}

	acmeCert.PrivateKey = keyBytes
	acmeCert.Certificate = certBytes
	return acmeCert, nil
}

func (c *Client) saveCertificate(certName, dnsNames string, certRes *certificate.Resource) (*AcmeCertificate, error) {
	expiryDate, err := getPEMCertExpiration(certRes.Certificate)
	if err != nil {
		return nil, fmt.Errorf("Failed to read certificate expiry date: %v", err)
	}
	serialNumber, err := getPEMCertSerialNumber(certRes.Certificate)
	if err != nil {
		return nil, fmt.Errorf("Failed to read certificate serial number: %v", err)
	}

	acmeCert := AcmeCertificate{
		Resource:     *certRes,
		ExpiryDate:   expiryDate,
		SerialNumber: serialNumber,
		DnsNames:     dnsNames,
	}

	certPath := c.CertPath(certName)
	maybeCreatePath(certPath)

	logrus.Debugf("Saving certificate '%s' to path '%s'", certName, certPath)

	certOut := path.Join(certPath, "fullchain.pem")
	privOut := path.Join(certPath, "privkey.pem")
	metaOut := path.Join(certPath, "metadata.json")

	err = ioutil.WriteFile(certOut, acmeCert.Certificate, 0600)
	if err != nil {
		return nil, fmt.Errorf("Failed to save certificate to '%s': %v", certOut, err)
	}

	logrus.Infof("Certificate saved to '%s'", certOut)

	err = ioutil.WriteFile(privOut, acmeCert.PrivateKey, 0600)
	if err != nil {
		return nil, fmt.Errorf("Failed to save private key to '%s': %v", privOut, err)
	}

	logrus.Infof("Private key saved to '%s'", privOut)

	jsonBytes, err := json.MarshalIndent(acmeCert, "", "\t")
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal meta data for certificate '%s': %v", certName, err)
	}

	err = ioutil.WriteFile(metaOut, jsonBytes, 0600)
	if err != nil {
		return nil, fmt.Errorf("Failed to save meta data to '%s': %v", metaOut, err)
	}

	return &acmeCert, nil
}

func (c *Client) ConfigPath() string {
	path := path.Join(StorageDir, strings.ToLower(string(c.apiVersion)))
	maybeCreatePath(path)
	return path
}

func (c *Client) ProviderName() string {
	return string(c.provider)
}

func (c *Client) ApiVersion() string {
	return string(c.apiVersion)
}

func (c *Client) CertPath(certName string) string {
	return path.Join(c.ConfigPath(), "certs", safeFileName(certName))
}

func dnsNamesIdentifier(domains []string) string {
	return strings.Join(domains, "|")
}

func maybeCreatePath(path string) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err = os.MkdirAll(path, 0700)
		if err != nil {
			logrus.Fatalf("Failed to create path: %v", err)
		}
	}
}

// safeFileName replaces separators with dashes and removes all
// characters other than alphanumerics, dashes, underscores and dots.
func safeFileName(str string) string {
	separators := regexp.MustCompile(`[ /&=+:]`)
	illegals := regexp.MustCompile(`[^[:alnum:]-_.]`)
	dashes := regexp.MustCompile(`[\-]+`)
	str = separators.ReplaceAllString(str, "-")
	str = illegals.ReplaceAllString(str, "")
	str = dashes.ReplaceAllString(str, "-")
	return str
}
