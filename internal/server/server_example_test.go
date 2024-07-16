package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"

	"github.com/sourcecd/monitoring/internal/retr"
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
	keyenc := ""
	ctx := context.Background()
	storage := storage.NewMemStorage()
	retrier := retr.NewRetr()
	mh := &metricHandlers{
		ctx:     ctx,
		storage: storage,
		rtr:     retrier,
	}

	srv := httptest.NewServer(chiRouter(mh, keyenc))
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
