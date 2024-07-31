package compression

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

var testBodyReq = "<html>TestRequest</html>"

func testGzipFunc(w http.ResponseWriter, r *http.Request) {
	b, _ := io.ReadAll(r.Body)
	defer r.Body.Close()
	w.WriteHeader(http.StatusOK)
	w.Write(b)
}

func BenchmarkCompress(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(testBodyReq))
		req.Header.Set("Accept-Encoding", "gzip")
		req.Header.Set("Content-Type", "text/html")
		ans := httptest.NewRecorder()
		b.StartTimer()

		GzipCompDecomp(testGzipFunc)(ans, req)

		b.StopTimer()
		res := ans.Result()
		if res.Header.Get("Content-Encoding") != "gzip" {
			b.Fatal("no compress")
		}
		body, _ := io.ReadAll(res.Body)
		res.Body.Close()
		if len(body) < 10 {
			b.Fatal("maybe no body")
		}
		b.StartTimer()
	}
}

func TestCompressionRead(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(testBodyReq))
	req.Header.Set("Accept-Encoding", "gzip")
	req.Header.Set("Content-Type", "text/html")
	ans := httptest.NewRecorder()

	GzipCompDecomp(testGzipFunc)(ans, req)

	res := ans.Result()
	require.Equal(t, "gzip", res.Header.Get("Content-Encoding"))
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	require.Less(t, 10, len(body))
}

func TestCompressionWrite(t *testing.T) {
	var b bytes.Buffer
	gz := gzip.NewWriter(&b)
	_, err := gz.Write([]byte(testBodyReq))
	require.NoError(t, err)
	err = gz.Close()
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewReader(b.Bytes()))
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("Content-Type", "text/html")
	ans := httptest.NewRecorder()

	GzipCompDecomp(testGzipFunc)(ans, req)

	res := ans.Result()
	body, _ := io.ReadAll(res.Body)
	res.Body.Close()
	require.Equal(t, testBodyReq, string(body))
}
