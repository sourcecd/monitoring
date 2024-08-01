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

	testCases := []struct {
		name,
		seckey,
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

	for _, v := range testCases {
		t.Run(v.name, func(t *testing.T) {
			ts := httptest.NewServer(SignCheck(testServerHTTPHandler, v.seckey))
			defer ts.Close()

			client := resty.New().R()
			testResp, err := client.SetBody(testBodyResp).Post(ts.URL)
			require.NoError(t, err)
			require.Equal(t, v.expresult, testResp.Header().Get(signHeaderType))
		})
	}
}

func TestServerReqSign(t *testing.T) {
	etalonSign := etalonHmacFunc(seckey, testBodyResp)

	testCases := []struct {
		Name       string
		TestHash   string
		StatusCode int
	}{
		{
			Name:       "correct hash",
			TestHash:   etalonSign,
			StatusCode: http.StatusOK,
		},
		{
			Name:       "not allowed hash",
			TestHash:   "f165b29bd896b6a9dcf5a0f3d5bac45cdbc96d0573a4b1fc1603d6a54acaa6d9",
			StatusCode: http.StatusBadRequest,
		},
		{
			Name:       "incorrect hash",
			TestHash:   "wronghash",
			StatusCode: http.StatusBadRequest,
		},
	}

	ts := httptest.NewServer(SignCheck(testServerHTTPHandler, seckey))
	defer ts.Close()

	for _, v := range testCases {
		t.Run(v.Name, func(t *testing.T) {
			client := resty.New().R()
			testResp, err := client.SetHeader("HashSHA256", v.TestHash).SetBody(testBodyResp).Post(ts.URL)
			require.NoError(t, err)
			require.Equal(t, v.StatusCode, testResp.StatusCode())
		})
	}
}
