package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/sourcecd/monitoring/internal/cryptandsign"
	"github.com/sourcecd/monitoring/internal/retrier"
	"github.com/sourcecd/monitoring/internal/storage"
)

func myError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func bodyRead(resp *http.Response, err error) []byte {
	myError(err)
	b, err := io.ReadAll(resp.Body)
	myError(err)
	defer resp.Body.Close()

	return b
}

func Example() {
	var keyenc, privkeypath, subnet string
	ctx := context.Background()
	storage := storage.NewMemStorage()
	reqRetrier := retrier.NewRetrier()
	mh := &metricHandlers{
		ctx:        ctx,
		storage:    storage,
		reqRetrier: reqRetrier,
		crypt:      cryptandsign.NewAsymmetricCryptRsa(),
	}

	srv := httptest.NewServer(chiRouter(mh, keyenc, privkeypath, subnet))
	defer srv.Close()
	client := srv.Client()
	// store metric value
	resp, err := client.Post(srv.URL+"/update/gauge/testgauge/0.1", "text/plain", nil)
	body := bodyRead(resp, err)
	fmt.Println(string(body))

	// get metric value
	resp, err = client.Get(srv.URL + "/value/gauge/testgauge")
	myError(err)
	body, err = io.ReadAll(resp.Body)
	myError(err)
	defer resp.Body.Close()

	fmt.Println(string(body))

	// Output:
	// OK
	// 0.1
}
