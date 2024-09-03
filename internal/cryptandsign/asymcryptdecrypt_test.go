package cryptandsign

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

var (
	resp       *resty.Response
	testString = "test_string"
)

func testSend(r *resty.Request, send, serverHost, xRealIp string) (*resty.Response, error) {
	resp, err := r.SetHeader("X-Real-IP", xRealIp).SetBody(send).Post(serverHost)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("ans: %d, %s", resp.StatusCode(), resp.Body())
	}
	return resp, err
}

func cryptHandlerFunc(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func TestAsymEncryptData(t *testing.T) {
	testIP := "::1"
	var (
		crypt AsymmetricCrypt
		err   error
	)
	crypt = NewAsymmetricCryptRsa()
	pubKey := "test_public.pem"
	srv := httptest.NewServer(http.HandlerFunc(cryptHandlerFunc))
	t.Cleanup(func() { srv.Close() })
	r := resty.New().R()

	snd := crypt.AsymmetricEncryptData(testSend, pubKey)
	resp, err = snd(r, testString, srv.URL, testIP)
	require.NoError(t, err)
}

func TestAsymDencryptData(t *testing.T) {
	var crypt AsymmetricCrypt = NewAsymmetricCryptRsa()
	privKey := "test_private.pem"
	w := httptest.NewRecorder()
	br := bytes.NewReader(resp.Body())
	r := httptest.NewRequest(http.MethodPost, "/", br)

	h := crypt.AsymmetricDencryptData(cryptHandlerFunc, privKey)
	h(w, r)
	res := w.Result()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, testString, string(b))
}
