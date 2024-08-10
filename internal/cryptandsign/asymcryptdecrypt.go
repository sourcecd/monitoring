package cryptandsign

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"hash"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/go-resty/resty/v2"
)

func encryptOAEPbyPart(hash hash.Hash, random io.Reader, public *rsa.PublicKey, msg []byte, label []byte) ([]byte, error) {
	msgLen := len(msg)
	step := public.Size() - 2*hash.Size() - 2
	var encryptedBytes []byte

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		encryptedBlockBytes, err := rsa.EncryptOAEP(hash, random, public, msg[start:finish], label)
		if err != nil {
			return nil, err
		}

		encryptedBytes = append(encryptedBytes, encryptedBlockBytes...)
	}

	return encryptedBytes, nil
}

func decryptOAEPbyPart(hash hash.Hash, random io.Reader, private *rsa.PrivateKey, msg []byte, label []byte) ([]byte, error) {
	msgLen := len(msg)
	step := private.PublicKey.Size()
	var decryptedBytes []byte

	for start := 0; start < msgLen; start += step {
		finish := start + step
		if finish > msgLen {
			finish = msgLen
		}

		decryptedBlockBytes, err := rsa.DecryptOAEP(hash, random, private, msg[start:finish], label)
		if err != nil {
			return nil, err
		}

		decryptedBytes = append(decryptedBytes, decryptedBlockBytes...)
	}

	return decryptedBytes, nil
}

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

			ciphertext, err := encryptOAEPbyPart(sha256.New(), rand.Reader, publicKey.(*rsa.PublicKey), []byte(send), []byte(""))
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

			ciphertextBase64, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "error read encrypted request", http.StatusBadRequest)
				return
			}
			ciphertext, err := base64.StdEncoding.DecodeString(string(ciphertextBase64))
			if err != nil {
				http.Error(w, "error decode base64 string", http.StatusBadRequest)
				return
			}

			plaintext, err := decryptOAEPbyPart(sha256.New(), nil, privateKey, ciphertext, []byte(""))
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
