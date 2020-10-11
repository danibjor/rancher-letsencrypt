package letsencrypt

import (
	"crypto"
	"encoding/pem"
	"io/ioutil"
	"os"
	"time"

	"github.com/go-acme/lego/v3/certcrypto"
)

func generatePrivateKey(keyType certcrypto.KeyType, file string) (crypto.PrivateKey, error) {
	privateKey, err := certcrypto.GeneratePrivateKey(keyType)
	if err != nil {
		return nil, err
	}

	certOut, err := os.Create(file)
	if err != nil {
		return nil, err
	}

	pemBlock := certcrypto.PEMBlock(privateKey)

	err = pem.Encode(certOut, pemBlock)

	certOut.Close()

	return privateKey, err
}

func loadPrivateKey(file string) (crypto.PrivateKey, error) {
	keyBytes, err := ioutil.ReadFile(file)
	if err != nil {
		return nil, err
	}

	return certcrypto.ParsePEMPrivateKey(keyBytes)
}

func getPEMCertExpiration(cert []byte) (time.Time, error) {
	pCert, err := certcrypto.ParsePEMCertificate(cert)
	if err != nil {
		return time.Time{}, err
	}

	return pCert.NotAfter, nil
}

func getPEMCertSerialNumber(cert []byte) (string, error) {
	pCert, err := certcrypto.ParsePEMCertificate(cert)
	if err != nil {
		return "", err
	}

	return pCert.SerialNumber.String(), nil
}
