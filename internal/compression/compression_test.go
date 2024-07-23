package compression

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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
