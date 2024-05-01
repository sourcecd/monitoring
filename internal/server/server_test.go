package server

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcecd/monitoring/internal/storage"
)

func TestUpdateHandler(t *testing.T) {
	type want struct {
		method     string
		statusCode int
		response   string
		request    string
	}

	testStorage := &storage.MemStorage{}
	testStorage.Setup()

	ts := httptest.NewServer(chiRouter(testStorage))
	defer ts.Close()

	testCase := []struct {
		name string
		want want
	}{
		{
			name: "test1",
			want: want{
				method:     http.MethodPost,
				statusCode: 200,
				response:   "OK",
				request:    "/update/counter/testCounter/100",
			},
		},
		{
			name: "test2",
			want: want{
				method:     http.MethodPost,
				statusCode: 200,
				response:   "OK",
				request:    "/update/gauge/testGauge/0.1",
			},
		},
		{
			name: "test3",
			want: want{
				method:     http.MethodPost,
				statusCode: 404,
				response:   "404 page not found\n",
				request:    "/update/gauge/testGauge",
			},
		},
		{
			name: "test4",
			want: want{
				method:     http.MethodPost,
				statusCode: 404,
				response:   "404 page not found\n",
				request:    "/update/counter/testcounter2",
			},
		},
		{
			name: "test5",
			want: want{
				method:     http.MethodPost,
				statusCode: 400,
				response:   "metric_type not found\n",
				request:    "/update/qwe/testGauge/0.1",
			},
		},
		{
			name: "test6-get-test1",
			want: want{
				method:     http.MethodGet,
				statusCode: 200,
				response:   "100\n",
				request:    "/value/counter/testCounter",
			},
		},
		{
			name: "test7-get-test2",
			want: want{
				method:     http.MethodGet,
				statusCode: 200,
				response:   "0.1\n",
				request:    "/value/gauge/testGauge",
			},
		},
	}

	for _, v := range testCase {
		t.Run(v.name, func(t *testing.T) {
			req, err := http.NewRequest(v.want.method, ts.URL+v.want.request, nil)
			req.Header.Set("Content-Type", "text/plain")
			require.NoError(t, err)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			require.Equal(t, v.want.statusCode, resp.StatusCode)
			require.Equal(t, v.want.response, string(body))
		})
	}
}

func TestUpdateHandlerJSON(t *testing.T) {
	type want struct {
		method      string
		statusCode  int
		response    string
		request     string
		requestBody string
	}

	testStorage := &storage.MemStorage{}
	testStorage.Setup()

	ts := httptest.NewServer(chiRouter(testStorage))
	defer ts.Close()

	//json api
	testCaseJSON := []struct {
		name string
		want want
	}{
		{
			name: "test1j",
			want: want{
				method:      http.MethodPost,
				statusCode:  200,
				response:    `{"id":"testCounter","type":"counter","delta":100}`,
				request:     "/update/",
				requestBody: `{"id": "testCounter", "type": "counter", "delta": 100}`,
			},
		},
		{
			name: "test2j",
			want: want{
				method:      http.MethodPost,
				statusCode:  200,
				response:    `{"id":"testGauge","type":"gauge","value":0.1}`,
				request:     "/update/",
				requestBody: `{"id": "testGauge", "type": "gauge", "value": 0.1}`,
			},
		},
		{
			name: "test3j",
			want: want{
				method:      http.MethodPost,
				statusCode:  400,
				response:    "no value of gauge metric",
				request:     "/update/",
				requestBody: `{"id": "testGauge", "type": "gauge"}`,
			},
		},
		{
			name: "test4j",
			want: want{
				method:      http.MethodPost,
				statusCode:  400,
				response:    "no value of counter metric",
				request:     "/update/",
				requestBody: `{"id": "testcounter2", "type": "counter"}`,
			},
		},
		{
			name: "test5j",
			want: want{
				method:      http.MethodPost,
				statusCode:  400,
				response:    "bad metric type",
				request:     "/update/",
				requestBody: `{"id": "testGauge", "type": "qwe", "value": 0.1}`,
			},
		},
		{
			name: "test6-get-test1j",
			want: want{
				method:      http.MethodPost,
				statusCode:  200,
				response:    `{"id":"testCounter","type":"counter","delta":100}`,
				request:     "/value/",
				requestBody: `{"id": "testCounter", "type": "counter"}`,
			},
		},
		{
			name: "test7-get-test2j",
			want: want{
				method:      http.MethodPost,
				statusCode:  200,
				response:    `{"id":"testGauge","type":"gauge","value":0.1}`,
				request:     "/value/",
				requestBody: `{"id": "testGauge", "type": "gauge"}`,
			},
		},
		{
			name: "test-get-fault1",
			want: want{
				method:      http.MethodPost,
				statusCode:  404,
				response:    "no value",
				request:     "/value/",
				requestBody: `{"id": "testGaugeNone", "type": "gauge"}`,
			},
		},
		{
			name: "test-get-fault2",
			want: want{
				method:      http.MethodPost,
				statusCode:  404,
				response:    "no value",
				request:     "/value/",
				requestBody: `{"id": "testGaugeNone", "type": "counter"}`,
			},
		},
		{
			name: "test-get-fault-metric-name",
			want: want{
				method:      http.MethodPost,
				statusCode:  404,
				response:    "bad metric type",
				request:     "/value/",
				requestBody: `{"id": "testGaugeNone", "type": "unk"}`,
			},
		},
	}

	//json api
	for _, v := range testCaseJSON {
		t.Run(v.name, func(t *testing.T) {
			var body []byte
			req, err := http.NewRequest(v.want.method, ts.URL+v.want.request, strings.NewReader(v.want.requestBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept-Encoding", "gzip")
			require.NoError(t, err)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			require.Equal(t, v.want.statusCode, resp.StatusCode)
			defer resp.Body.Close()

			compressType := resp.Header.Get("Content-Encoding")
			require.Equal(t, "gzip", compressType)

			gzr, err := gzip.NewReader(resp.Body)
			require.NoError(t, err)

			body, err = io.ReadAll(gzr)
			require.NoError(t, err)

			require.Equal(t, v.want.response, strings.Trim(string(body), "\n"))
		})
	}
}
