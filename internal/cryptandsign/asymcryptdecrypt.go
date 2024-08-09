package cryptandsign

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-resty/resty/v2"
)

func AsymEncryptData(s AgentSendFunc, pubkeypath string) AgentSendFunc {
	return func(r *resty.Request, send, serverHost string) (*resty.Response, error) {
		if pubkeypath != "" {
			publicKeyPEM, err := os.ReadFile(pubkeypath)
			if err != nil {
				log.Fatal("error reading pub key:", err)
			}
			publicKeyBlock, _ := pem.Decode(publicKeyPEM)
			publicKey, err := x509.ParsePKIXPublicKey(publicKeyBlock.Bytes)
			if err != nil {
				log.Fatal("error parse pub key:", err)
			}

			ciphertext, err := rsa.EncryptPKCS1v15(rand.Reader, publicKey.(*rsa.PublicKey), []byte(send))
			if err != nil {
				return nil, err
			}

			return s(r, base64.StdEncoding.EncodeToString(ciphertext), serverHost)
		}

		return s(r, send, serverHost)
	}
}

func AsymDencryptData(h http.HandlerFunc, privkeypath string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if privkeypath != "" {
			privateKeyPEM, err := os.ReadFile(privkeypath)
			if err != nil {
				log.Fatal("error reading priv key:", err)
			}
			privateKeyBlock, _ := pem.Decode(privateKeyPEM)
			privateKey, err := x509.ParsePKCS1PrivateKey(privateKeyBlock.Bytes)
			if err != nil {
				log.Fatal("error parse priv key:", err)
			}

			ciphertext, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "error read encrypted request", http.StatusBadRequest)
				return
			}

			plaintext, err := rsa.DecryptPKCS1v15(rand.Reader, privateKey, ciphertext)
			if err != nil {
				http.Error(w, "error data dencryption", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(plaintext))
			defer r.Body.Close()
		}

		h(w, r)
	}
}
