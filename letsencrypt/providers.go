package letsencrypt

import (
	"fmt"
	"os"

	"github.com/go-acme/lego/v3/challenge"
	"github.com/go-acme/lego/v3/challenge/http01"
	"github.com/go-acme/lego/v3/providers/dns/auroradns"
	"github.com/go-acme/lego/v3/providers/dns/cloudflare"
	"github.com/go-acme/lego/v3/providers/dns/digitalocean"
	"github.com/go-acme/lego/v3/providers/dns/dyn"
	"github.com/go-acme/lego/v3/providers/dns/gandi"
	"github.com/go-acme/lego/v3/providers/dns/ns1"
	"github.com/go-acme/lego/v3/providers/dns/ovh"
	"github.com/go-acme/lego/v3/providers/dns/route53"
	"github.com/go-acme/lego/v3/providers/dns/vultr"
)

// ProviderOpts is used to configure the DNS provider
// used by the Let's Encrypt client for domain validation
type ProviderOpts struct {
	Provider Provider

	// Aurora credentials
	AuroraUserId   string
	AuroraKey      string
	AuroraEndpoint string

	// AWS Route 53 credentials
	AwsAccessKey string
	AwsSecretKey string

	// CloudFlare credentials
	CfApiEmail     string
	CfApiKey       string
	CfDnsApiToken  string
	CfZoneApiToken string

	// DigitalOcean credentials
	DoAccessToken string

	// Dyn credentials
	DynCustomerName string
	DynUserName     string
	DynPassword     string

	// Gandi credentials
	GandiApiKey string

	// NS1 credentials
	NS1ApiKey string

	// OVH credentials
	OvhEndpoint          string
	OvhApplicationKey    string
	OvhApplicationSecret string
	OvhConsumerKey       string

	// Vultr credentials
	VultrApiKey string
}

type Provider string

const (
	AURORA       = Provider("Aurora")
	CLOUDFLARE   = Provider("CloudFlare")
	DIGITALOCEAN = Provider("DigitalOcean")
	DYN          = Provider("Dyn")
	GANDI        = Provider("Gandi")
	NS1          = Provider("NS1")
	OVH          = Provider("Ovh")
	ROUTE53      = Provider("Route53")
	VULTR        = Provider("Vultr")
	HTTP         = Provider("HTTP")
)

type ProviderFactory struct {
	factory   interface{}
	challenge challenge.Type
}

var providerFactory = map[Provider]ProviderFactory{
	AURORA:       {makeAuroraProvider, challenge.DNS01},
	CLOUDFLARE:   {makeCloudflareProvider, challenge.DNS01},
	DIGITALOCEAN: {makeDigitalOceanProvider, challenge.DNS01},
	DYN:          {makeDynProvider, challenge.DNS01},
	GANDI:        {makeGandiProvider, challenge.DNS01},
	NS1:          {makeNS1Provider, challenge.DNS01},
	OVH:          {makeOvhProvider, challenge.DNS01},
	ROUTE53:      {makeRoute53Provider, challenge.DNS01},
	VULTR:        {makeVultrProvider, challenge.DNS01},
	HTTP:         {makeHTTPProvider, challenge.HTTP01},
}

func getProvider(opts ProviderOpts) (challenge.Provider, challenge.Type, error) {
	if f, ok := providerFactory[opts.Provider]; ok {
		provider, err := f.factory.(func(ProviderOpts) (challenge.Provider, error))(opts)
		if err != nil {
			return nil, f.challenge, err
		}
		return provider, f.challenge, nil
	}
	irrelevant := challenge.DNS01
	return nil, irrelevant, fmt.Errorf("Unsupported provider: %s", opts.Provider)
}

// returns a preconfigured Aurora challenge.Provider
func makeAuroraProvider(opts ProviderOpts) (challenge.Provider, error) {
	if len(opts.AuroraUserId) == 0 {
		return nil, fmt.Errorf("Aurora User Id is not set")
	}

	if len(opts.AuroraKey) == 0 {
		return nil, fmt.Errorf("Aurora Key is not set")
	}

	endpoint := opts.AuroraEndpoint
	if len(endpoint) == 0 {
		endpoint = "https://api.auroradns.eu"
	}

	config := auroradns.NewDefaultConfig()
	config.BaseURL = endpoint
	config.UserID = opts.AuroraUserId
	config.Key = opts.AuroraKey

	provider, err := auroradns.NewDNSProviderConfig(config)

	if err != nil {
		return nil, err
	}

	return provider, nil
}

// returns a preconfigured CloudFlare challenge.Provider
func makeCloudflareProvider(opts ProviderOpts) (challenge.Provider, error) {
	if len(opts.CfDnsApiToken) == 0 {
		if len(opts.CfApiEmail) == 0 {
			return nil, fmt.Errorf("CloudFlare email is not set")
		}
		if len(opts.CfApiKey) == 0 {
			return nil, fmt.Errorf("CloudFlare API key is not set")
		}
	}

	config := cloudflare.NewDefaultConfig()
	config.AuthEmail = opts.CfApiEmail
	config.AuthKey = opts.CfApiKey
	config.AuthToken = opts.CfDnsApiToken
	config.ZoneToken = opts.CfZoneApiToken

	provider, err := cloudflare.NewDNSProviderConfig(config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// returns a preconfigured DigitalOcean challenge.Provider
func makeDigitalOceanProvider(opts ProviderOpts) (challenge.Provider, error) {
	if len(opts.DoAccessToken) == 0 {
		return nil, fmt.Errorf("DigitalOcean API access token is not set")
	}

	config := digitalocean.NewDefaultConfig()
	config.AuthToken = opts.DoAccessToken

	provider, err := digitalocean.NewDNSProviderConfig(config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// returns a preconfigured Route53 challenge.Provider
func makeRoute53Provider(opts ProviderOpts) (challenge.Provider, error) {
	if len(opts.AwsAccessKey) != 0 {
		os.Setenv("AWS_ACCESS_KEY_ID", opts.AwsAccessKey)
	}
	if len(opts.AwsSecretKey) != 0 {
		os.Setenv("AWS_SECRET_ACCESS_KEY", opts.AwsSecretKey)
	}

	os.Setenv("AWS_REGION", "us-east-1")

	provider, err := route53.NewDNSProvider()
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// returns a preconfigured Dyn challenge.Provider
func makeDynProvider(opts ProviderOpts) (challenge.Provider, error) {
	if len(opts.DynCustomerName) == 0 {
		return nil, fmt.Errorf("Dyn customer name is not set")
	}
	if len(opts.DynUserName) == 0 {
		return nil, fmt.Errorf("Dyn user name is not set")
	}
	if len(opts.DynPassword) == 0 {
		return nil, fmt.Errorf("Dyn password is not set")
	}

	config := dyn.NewDefaultConfig()
	config.CustomerName = opts.DynCustomerName
	config.UserName = opts.DynUserName
	config.Password = opts.DynPassword

	provider, err := dyn.NewDNSProviderConfig(config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// returns a preconfigured Vultr challenge.Provider
func makeVultrProvider(opts ProviderOpts) (challenge.Provider, error) {
	if len(opts.VultrApiKey) == 0 {
		return nil, fmt.Errorf("Vultr API key is not set")
	}

	config := vultr.NewDefaultConfig()
	config.APIKey = opts.VultrApiKey

	provider, err := vultr.NewDNSProviderConfig(config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// returns a preconfigured Ovh challenge.Provider
func makeOvhProvider(opts ProviderOpts) (challenge.Provider, error) {
	if len(opts.OvhApplicationKey) == 0 {
		return nil, fmt.Errorf("OVH application key is not set")
	}
	if len(opts.OvhApplicationSecret) == 0 {
		return nil, fmt.Errorf("OVH application secret is not set")
	}
	if len(opts.OvhConsumerKey) == 0 {
		return nil, fmt.Errorf("OVH consumer key is not set")
	}

	config := ovh.NewDefaultConfig()
	config.ApplicationKey = opts.OvhApplicationKey
	config.ApplicationSecret = opts.OvhApplicationSecret
	config.ConsumerKey = opts.OvhConsumerKey
	if len(opts.OvhEndpoint) == 0 {
		config.APIEndpoint = "ovh-eu"
	} else {
		config.APIEndpoint = opts.OvhEndpoint
	}

	provider, err := ovh.NewDNSProviderConfig(config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// returns a preconfigured Gandi challenge.Provider
func makeGandiProvider(opts ProviderOpts) (challenge.Provider, error) {
	if len(opts.GandiApiKey) == 0 {
		return nil, fmt.Errorf("Gandi API key is not set")
	}

	config := gandi.NewDefaultConfig()
	config.APIKey = opts.GandiApiKey

	provider, err := gandi.NewDNSProviderConfig(config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}

// returns a preconfigured HTTP challenge.Provider
func makeHTTPProvider(opts ProviderOpts) (challenge.Provider, error) {
	provider := http01.NewProviderServer("", "")
	return provider, nil
}

// returns a preconfigured NS1 challenge.Provider
func makeNS1Provider(opts ProviderOpts) (challenge.Provider, error) {
	if len(opts.NS1ApiKey) == 0 {
		return nil, fmt.Errorf("NS1 API key is not set")
	}

	config := ns1.NewDefaultConfig()
	config.APIKey = opts.NS1ApiKey

	provider, err := ns1.NewDNSProviderConfig(config)
	if err != nil {
		return nil, err
	}
	return provider, nil
}
