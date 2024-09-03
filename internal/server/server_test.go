package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"html/template"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/sourcecd/monitoring/internal/cryptandsign"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/retrier"
	"github.com/sourcecd/monitoring/internal/storage"
	"github.com/sourcecd/monitoring/mocks"
)

func TestUpdateHandler(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	reqRetrier := retrier.NewRetrier()

	var keyenc, privkeypath string
	type want struct {
		method     string
		response   string
		request    string
		statusCode int
	}

	testStorage := storage.NewMemStorage()

	mh := &metricHandlers{
		ctx:        ctx,
		storage:    testStorage,
		reqRetrier: reqRetrier,
		crypt:      cryptandsign.NewAsymmetricCryptRsa(),
	}

	ts := httptest.NewServer(chiRouter(mh, keyenc, privkeypath, nil))
	t.Cleanup(func() { ts.Close() })

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
	t.Parallel()
	ctx := context.Background()
	reqRetrier := retrier.NewRetrier()

	var keyenc, privkeypath string
	type want struct {
		method      string
		response    string
		request     string
		requestBody string
		statusCode  int
	}

	testStorage := storage.NewMemStorage()

	mh := &metricHandlers{
		ctx:        ctx,
		storage:    testStorage,
		reqRetrier: reqRetrier,
		crypt:      cryptandsign.NewAsymmetricCryptRsa(),
	}

	ts := httptest.NewServer(chiRouter(mh, keyenc, privkeypath, nil))
	t.Cleanup(func() { ts.Close() })

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
				response:    `{"delta":100,"id":"testCounter","type":"counter"}`,
				request:     "/update/",
				requestBody: `{"id": "testCounter", "type": "counter", "delta": 100}`,
			},
		},
		{
			name: "test2j",
			want: want{
				method:      http.MethodPost,
				statusCode:  200,
				response:    `{"value":0.1,"id":"testGauge","type":"gauge"}`,
				request:     "/update/",
				requestBody: `{"id": "testGauge", "type": "gauge", "value": 0.1}`,
			},
		},
		{
			name: "test3j",
			want: want{
				method:      http.MethodPost,
				statusCode:  400,
				response:    "bad metric type or no metric value or id is empty",
				request:     "/update/",
				requestBody: `{"id": "testGauge", "type": "gauge"}`,
			},
		},
		{
			name: "test4j",
			want: want{
				method:      http.MethodPost,
				statusCode:  400,
				response:    "bad metric type or no metric value or id is empty",
				request:     "/update/",
				requestBody: `{"id": "testcounter2", "type": "counter"}`,
			},
		},
		{
			name: "test5j",
			want: want{
				method:      http.MethodPost,
				statusCode:  400,
				response:    "bad metric type or no metric value or id is empty",
				request:     "/update/",
				requestBody: `{"id": "testGauge", "type": "qwe", "value": 0.1}`,
			},
		},
		{
			name: "test6-get-test1j",
			want: want{
				method:      http.MethodPost,
				statusCode:  200,
				response:    `{"delta":100,"id":"testCounter","type":"counter"}`,
				request:     "/value/",
				requestBody: `{"id": "testCounter", "type": "counter"}`,
			},
		},
		{
			name: "test7-get-test2j",
			want: want{
				method:      http.MethodPost,
				statusCode:  200,
				response:    `{"value":0.1,"id":"testGauge","type":"gauge"}`,
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

func TestDB(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	reqRetrier := retrier.NewRetrier()

	var keyenc, privkeypath string
	ctrl := gomock.NewController(t)
	t.Cleanup(func() { ctrl.Finish() })

	mDB := mocks.NewMockStoreMetrics(ctrl)

	mh := &metricHandlers{
		ctx:        ctx,
		storage:    mDB,
		reqRetrier: reqRetrier,
		crypt:      cryptandsign.NewAsymmetricCryptRsa(),
	}

	ts := httptest.NewServer(chiRouter(mh, keyenc, privkeypath, nil))
	t.Cleanup(func() { ts.Close() })

	gomock.InOrder(
		mDB.EXPECT().Ping(gomock.Any()).Return(nil),
		mDB.EXPECT().Ping(gomock.Any()).Return(errors.New("Connection refused")),
	)
	testPingCases := []struct {
		mockAns       error
		name          string
		expAns        string
		expStatusCode int
	}{
		{
			name:          "PingOK",
			expStatusCode: http.StatusOK,
			expAns:        "OK\n",
		},
		{
			name:          "PingWrong",
			expStatusCode: http.StatusInternalServerError,
			expAns:        "Connection refused\n",
		},
	}

	for _, v := range testPingCases {
		t.Run(v.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, ts.URL+"/ping", nil)
			require.NoError(t, err)

			resp, err := ts.Client().Do(req)
			require.NoError(t, err)
			defer resp.Body.Close()

			b, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			require.Equal(t, v.expStatusCode, resp.StatusCode)
			require.Equal(t, string(b), v.expAns)
		})
	}

}

func TestGetAll(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	tmpl, _ := template.New("data").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8" />
	<meta name="counters" content="width=device-width, initial-scale=1.0" />
	<title>Counters</title>
</head>
<body>
<pre>
{{ .}}
</pre>
</body>
</html>`)

	expectedTestData := `---Counters---
---Gauge---
`
	ctx := context.Background()
	storage := storage.NewMemStorage()
	reqRetrier := retrier.NewRetrier()
	mh := metricHandlers{
		ctx:        ctx,
		storage:    storage,
		reqRetrier: reqRetrier,
		crypt:      cryptandsign.NewAsymmetricCryptRsa(),
	}
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	// GetAll function
	testHandleFunc := mh.getAll()
	testHandleFunc(response, request)

	resp := response.Result()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	defer resp.Body.Close()
	_ = tmpl.Execute(&buf, expectedTestData)
	require.Equal(t, buf.String(), string(body))
}

func TestUpdateBatchMetricsJSON(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	storage := storage.NewMemStorage()
	reqRetrier := retrier.NewRetrier()
	mh := metricHandlers{
		ctx:        ctx,
		storage:    storage,
		reqRetrier: reqRetrier,
		crypt:      cryptandsign.NewAsymmetricCryptRsa(),
	}
	testRequest := `[{"type": "gauge", "id": "testmetric", "value": 0.1}, {"type": "counter", "id": "testmetric2", "delta": 1}]`

	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(testRequest))
	request.Header.Set("Content-Type", "application/json")

	testHandleFunc := mh.updateBatchMetricsJSON()
	testHandleFunc(response, request)

	res := response.Result()
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)
	ifaceG, err := storage.GetMetric(ctx, "gauge", "testmetric")
	require.NoError(t, err)
	require.Equal(t, metrictypes.Gauge(0.1), ifaceG.(metrictypes.Gauge))
	ifaceC, err := storage.GetMetric(ctx, "counter", "testmetric2")
	require.NoError(t, err)
	require.Equal(t, metrictypes.Counter(1), ifaceC.(metrictypes.Counter))
}

func TestSaveToFile(t *testing.T) {
	f, err := os.CreateTemp("", "save-mon-test")
	require.NoError(t, err)
	defer f.Close()
	t.Cleanup(func() { os.Remove(f.Name()) })
	m := storage.NewMemStorage()
	saveToFile(m, f.Name(), 0)
}
