package cryptandsign

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"log"
	"net/http"

	"github.com/go-resty/resty/v2"
)

const signHeaderType = "HashSHA256"

type (
	AgentSendFunc func(r *resty.Request, send, serverHost string) (*resty.Response, error)

	responseData struct {
		respCode int
	}

	signResponseWriter struct {
		wr              http.ResponseWriter
		respData        *responseData
		sign            hash.Hash
		headersSetDone  chan bool
		headersSendDone chan bool
	}
)

func (s *signResponseWriter) Write(b []byte) (int, error) {
	s.sign.Write(b)
	res := s.sign.Sum(nil)
	s.wr.Header().Set(signHeaderType, hex.EncodeToString(res))
	s.headersSetDone <- true
	<-s.headersSendDone
	size, err := s.wr.Write(b)
	return size, err
}

func (s *signResponseWriter) WriteHeader(statusCode int) {
	go func() {
		<-s.headersSetDone
		s.wr.WriteHeader(statusCode)
		s.headersSendDone <- true
	}()
	s.respData.respCode = statusCode
}

func (s *signResponseWriter) Header() http.Header {
	return s.wr.Header()
}

func SignCheck(h http.HandlerFunc, seckey string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hashSignStr := r.Header.Get(signHeaderType)
		if seckey == "" {
			h(w, r)
			return
		}

		hm := hmac.New(sha256.New, []byte(seckey))

		rdata := &responseData{}
		cw := &signResponseWriter{
			wr:              w,
			respData:        rdata,
			sign:            hm,
			headersSetDone:  make(chan bool, 1),
			headersSendDone: make(chan bool, 1)}

		// for hmac
		if hashSignStr != "" {
			req, err := io.ReadAll(r.Body)
			if err != nil {
				log.Println("sign: error read request body")
				http.Error(w, "error read request body", http.StatusBadRequest)
				return
			}
			r.Body = io.NopCloser(bytes.NewBuffer(req))
			defer r.Body.Close()
			hashSign, err := hex.DecodeString(hashSignStr)
			if err != nil {
				log.Println("sign: can't decode hashSign")
				http.Error(w, "can't decode hashSign", http.StatusBadRequest)
				return
			}
			hm.Write(req)
			reshm := hm.Sum(nil)
			if hmac.Equal(hashSign, reshm) {
				h(cw, r)
				return
			}
			log.Println("sign: sign error")
			http.Error(w, "sign error", http.StatusBadRequest)
			return
		}
		h(cw, r)
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
