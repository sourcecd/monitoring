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

const seckey = "Kaib8eel"

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

func TestAgentSign(t *testing.T) {
	myTestBody := `
	Test body need to sign
	`

	//Etalon gen sign !
	h := hmac.New(sha256.New, []byte(seckey))
	h.Write([]byte(myTestBody))
	etalonSign := hex.EncodeToString(h.Sum(nil))

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
			resp, err := testSendF(rReq, myTestBody, ts.URL)
			require.NoError(t, err)
			respSign := resp.Header().Get(signHeaderType)
			require.Equal(t, v.expresult, respSign)
		})
	}
}
