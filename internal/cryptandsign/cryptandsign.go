package cryptandsign

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"net/http"

	"github.com/go-resty/resty/v2"
)

const signHeaderType = "HashSHA256"

type AgentSendFunc func(r *resty.Request, send, serverHost string) (*resty.Response, error)

func SignCheck(h http.HandlerFunc, seckey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if seckey == "" {
			h(w, r)
			return
		}
		hashSignStr := r.Header.Get(signHeaderType)
		if hashSignStr == "" {
			// tmp
			log.Panic("2")
			http.Error(w, "no hashSign in headers", http.StatusBadRequest)
			return
		}
		// for hmac
		req, err := io.ReadAll(r.Body)
		if err != nil {
			// tmp
			log.Panic("3")
			http.Error(w, "error read request body", http.StatusBadRequest)
			return
		}
		r.Body = io.NopCloser(bytes.NewBuffer(req))
		hashSign, err := hex.DecodeString(hashSignStr)
		if err != nil {
			// tmp
			log.Panic("4")
			http.Error(w, "can't decode hashSign", http.StatusBadRequest)
			return
		}
		hm := hmac.New(sha256.New, []byte(seckey))
		hm.Write(req)
		reshm := hm.Sum(nil)
		if hmac.Equal(hashSign, reshm) {
			h(w, r)
			return
		}
		// tmp
		log.Panic("5")
		http.Error(w, "sign error", http.StatusBadRequest)
	}
}

func SignNew(s AgentSendFunc, seckey string) AgentSendFunc {
	return func(r *resty.Request, send, serverHost string) (*resty.Response, error) {
		if seckey == "" {
			return s(r, send, serverHost)
		}
		hm := hmac.New(sha256.New, []byte(seckey))
		hm.Write([]byte(send))
		reshm := hm.Sum(nil)
		reshmstr := hex.EncodeToString(reshm)
		r.Header.Set(signHeaderType, reshmstr)
		return s(r, send, serverHost)
	}
}
