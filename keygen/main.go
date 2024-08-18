package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"log"
	"os"
)

func genPrivatePub() ([]byte, *rsa.PublicKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, nil, err
	}

	publicKey := &privateKey.PublicKey

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	return privateKeyPEM, publicKey, nil
}

func marshalPubKey(publicKey *rsa.PublicKey) ([]byte, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return nil, err
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	return publicKeyPEM, nil
}

func main() {
	privateKey, publicKey, err := genPrivatePub()
	if err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile("private.pem", privateKey, 0644); err != nil {
		log.Fatal(err)
	}

	publicKeyPEM, err := marshalPubKey(publicKey)
	if err != nil {
		log.Fatal(err)
	}
	if err = os.WriteFile("public.pem", publicKeyPEM, 0644); err != nil {
		log.Fatal(err)
	}
}
