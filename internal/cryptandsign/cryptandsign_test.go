package cryptandsign

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-resty/resty/v2"
	"github.com/stretchr/testify/require"
)

const (
	seckey       = "Kaib8eel"
	testBodyReq  = "Test body need to sign"
	testBodyResp = "Test body need to sign server ans"
)

func testServerHTTPHandler(w http.ResponseWriter, r *http.Request) {
	signHeader := r.Header.Get(signHeaderType)
	allBody, _ := io.ReadAll(r.Body)
	if r.Header.Get(signHeaderType) != "" {
		w.Header().Set(signHeaderType, signHeader)
	}
	w.WriteHeader(http.StatusOK)
	w.Write(allBody)
}

func testSendFunc(r *resty.Request, send, serverHost string) (*resty.Response, error) {
	return r.SetBody(send).Post(serverHost)
}

func etalonHmacFunc(seckey, body string) string {
	h := hmac.New(sha256.New, []byte(seckey))
	h.Write([]byte(body))
	return hex.EncodeToString(h.Sum(nil))
}

func TestAgentSign(t *testing.T) {
	//Etalon gen sign !
	etalonSign := etalonHmacFunc(seckey, testBodyReq)

	testCases := []struct {
		name      string
		seckey    string
		expresult string
	}{
		{
			name:      "with_key",
			seckey:    seckey,
			expresult: etalonSign,
		},
		{
			name:      "without_key",
			seckey:    "",
			expresult: "",
		},
	}

	ts := httptest.NewServer(http.HandlerFunc(testServerHTTPHandler))
	defer ts.Close()

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			client := resty.New()
			rReq := client.R()

			// Check Main Sign Func
			testSendF := SignNew(testSendFunc, v.seckey)
			resp, err := testSendF(rReq, testBodyReq, ts.URL)
			require.NoError(t, err)
			respSign := resp.Header().Get(signHeaderType)
			require.Equal(t, v.expresult, respSign)
		})
	}
}

func TestServerRespSign(t *testing.T) {
	etalonSign := etalonHmacFunc(seckey, testBodyResp)

	ts := httptest.NewServer(SignCheck(testServerHTTPHandler, seckey))
	defer ts.Close()

	client := resty.New()
	testResp, err := client.R().SetBody(testBodyResp).Post(ts.URL)
	require.NoError(t, err)
	require.Equal(t, etalonSign, testResp.Header().Get(signHeaderType))
}
