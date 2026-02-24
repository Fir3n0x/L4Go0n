package cmd


import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/x509"
	"encoding/pem"
	"crypto/rand"
	"os"
)

func loadOrGenerateKey(filePath string) (*ecdsa.PrivateKey, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Generate a new key if the file does not exist
		privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
		if err != nil {
			return nil, err
		}

		// Serialize the key to PEM format
		der, _ := x509.MarshalECPrivateKey(privateKey)
		pemBlock := &pem.Block{
			Type:  "EC PRIVATE KEY",
			Bytes: der,
		}
		pemData := pem.EncodeToMemory(pemBlock)
		err = os.WriteFile(filePath, pemData, 0600)
		return privateKey, err
	}

	// Load the existing key
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	return x509.ParseECPrivateKey(block.Bytes)
}
