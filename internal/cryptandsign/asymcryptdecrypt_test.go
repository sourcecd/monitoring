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

func testSend(r *resty.Request, send, serverHost string) (*resty.Response, error) {
	resp, err := r.SetBody(send).Post(serverHost)
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
	var err error
	pubKey := "test_public.pem"
	srv := httptest.NewServer(http.HandlerFunc(cryptHandlerFunc))
	t.Cleanup(func() { srv.Close() })
	r := resty.New().R()

	snd := AsymEncryptData(testSend, pubKey)
	resp, err = snd(r, testString, srv.URL)
	require.NoError(t, err)
}

func TestAsymDencryptData(t *testing.T) {
	privKey := "test_private.pem"
	w := httptest.NewRecorder()
	br := bytes.NewReader(resp.Body())
	r := httptest.NewRequest(http.MethodPost, "/", br)

	h := AsymDencryptData(cryptHandlerFunc, privKey)
	h(w, r)
	res := w.Result()
	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, testString, string(b))
}
