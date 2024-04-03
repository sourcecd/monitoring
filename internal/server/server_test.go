package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sourcecd/monitoring/internal/storage"
)

func TestUpdateHandler(t *testing.T) {
	type want struct {
		statusCode int
		response   string
		request    string
		metric     string
		mType      string
	}
	
	testStorage := &storage.MemStorage{}
	testStorage.Setup()

	hndl := updateMetrics(testStorage)

	testCase := []struct {
		name string
		want want
	}{
		{
			name: "test1",
			want: want{
				statusCode: 200,
				response:   "OK",
				request:    "/update/counter/testCounter/100",
				metric:     "testCounter",
				mType:      "counter",
			},
		},
		{
			name: "test2",
			want: want{
				statusCode: 200,
				response:   "OK",
				request:    "/update/gauge/testGauge/0.1",
				metric:     "testGauge",
				mType:      "gauge",
			},
		},
	}

	for _, v := range testCase {
		t.Run(v.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, v.want.request, nil)
			resp := httptest.NewRecorder()
			hndl(resp, req)
			respParsed := resp.Result()

			body, _ := io.ReadAll(respParsed.Body)
			defer respParsed.Body.Close()

			assert.Equal(t, v.want.statusCode, respParsed.StatusCode)
			assert.Equal(t, v.want.response, string(body))
			if v.want.mType == "counter" {
				_, err := testStorage.GetCounter(v.want.metric)
				assert.NoError(t, err)
			} else if v.want.mType == "gauge" {
				_, err := testStorage.GetGauge(v.want.metric)
				assert.NoError(t, err)
			} else {
				t.Error("unknown metric type")
			}
		})
	}
}
