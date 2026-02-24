package cmd

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

var ServerPublicKey *ecdsa.PublicKey

func LoadOrGenerateKey(filePath string) (*ecdsa.PrivateKey, error) {
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
		if err != nil {
			return nil, err
		}

		return privateKey, err
	}

	// Load the existing key
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(data)
	if block == nil || block.Type != "EC PRIVATE KEY" {
        return nil, fmt.Errorf("invalid PEM block type: %v", block.Type)
    }
	
	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
    if err != nil {
        return nil, err
    }

    return privateKey, nil
}

func ExportPublicKeyToPEM(pub *ecdsa.PublicKey) []byte {
    der, err := x509.MarshalPKIXPublicKey(pub)
    if err != nil {
        return nil
    }
    return pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: der})
}
