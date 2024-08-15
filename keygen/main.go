package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"
)

func genPrivatePub() ([]byte, *rsa.PublicKey) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		panic(err)
	}

	publicKey := &privateKey.PublicKey

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	return privateKeyPEM, publicKey
}

func marshalPubKey(publicKey *rsa.PublicKey) []byte {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	return publicKeyPEM
}

func main() {
	privateKey, publicKey := genPrivatePub()
	err := os.WriteFile("private.pem", privateKey, 0644)
	if err != nil {
		panic(err)
	}

	publicKeyPEM := marshalPubKey(publicKey)
	err = os.WriteFile("public.pem", publicKeyPEM, 0644)
	if err != nil {
		panic(err)
	}
}
