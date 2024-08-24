package agent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/stretchr/testify/require"

	"github.com/sourcecd/monitoring/internal/cryptandsign"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
)

func testServerHTTPHandler(w http.ResponseWriter, r *http.Request) {
	allBody, _ := io.ReadAll(r.Body)
	w.WriteHeader(http.StatusOK)
	w.Write(allBody)
}

func TestMetricsAgent(t *testing.T) {
	t.Parallel()
	metrics := &jsonModelsMetrics{}
	rtm := &MemStats{}
	sysMetrics := &sysMon{}
	numCount := 1
	randVal := 0
	testAllocSens := 0
	testrandVal := metrictypes.Gauge(0.123)
	testPollCount := metrictypes.Counter(5)
	testRtFiledName := "Alloc"
	expMetricURLs := `[{"id":"RandomValue","type":"gauge","value":0.123}, 
	{"id":"PollCount","type":"counter","delta":5}, {"id":"Alloc","type":"gauge","value":0}]`
	// get val
	m := reflect.ValueOf(rtm).Elem()
	stringTestRt := fmt.Sprintf("%v", m.FieldByName(testRtFiledName))
	fl64, err := strconv.ParseFloat(fmt.Sprintf("%v", stringTestRt), 64)
	require.NoError(t, err)

	updateMetrics(rtm, sysMetrics)

	metrics.jsonMetricsSlice = append(metrics.jsonMetricsSlice, models.Metrics{ID: "RandomValue", MType: metrictypes.GaugeType, Value: (*float64)(&testrandVal)})
	metrics.jsonMetricsSlice = append(metrics.jsonMetricsSlice, models.Metrics{ID: "PollCount", MType: metrictypes.CounterType, Delta: (*int64)(&testPollCount)})
	metrics.jsonMetricsSlice = append(metrics.jsonMetricsSlice, models.Metrics{ID: "Alloc", MType: metrictypes.GaugeType, Value: &fl64})

	jres, err := encodeJSON(metrics)
	require.NoError(t, err)
	require.JSONEq(t, jres, expMetricURLs)

	require.Equal(t, metrictypes.Counter(numCount), sysMetrics.pollCount)
	require.NotEqual(t, metrictypes.Gauge(randVal), sysMetrics.randomValue)
	require.NotEqual(t, uint64(testAllocSens), rtm.Alloc)
}

func TestUpdateSysKernMetrics(t *testing.T) {
	t.Parallel()
	cpuCount, _ := cpu.Counts(true)
	m := &kernelMetrics{CPUutilization: make([]metrictypes.Gauge, cpuCount)}
	updateSysKernMetrics(m)
	require.Greater(t, m.FreeMemory, metrictypes.Gauge(0))
	require.Greater(t, m.TotalMemory, metrictypes.Gauge(0))
	require.Greater(t, len(m.CPUutilization), 0)
}

func TestSendFunc(t *testing.T) {
	t.Parallel()
	testIP := "::1"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, r.Method, "POST")

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.Equal(t, string(body), "testRequest")
		defer r.Body.Close()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("testRequestDone"))
	}))
	t.Cleanup(func() { srv.Close() })

	client := resty.New()
	request := client.R()

	// send func
	response, err := send(request, "testRequest", srv.URL, testIP)
	require.NoError(t, err)
	require.Equal(t, "testRequestDone", string(response.Body()))
	require.Equal(t, http.StatusOK, response.StatusCode())
}

func TestParseRtm(t *testing.T) {
	t.Parallel()
	m := &MemStats{}
	jsonMetrics := &jsonModelsMetrics{}
	sysMon := &sysMon{}

	runtime.ReadMemStats(&m.MemStats)

	// parseRtm func
	parseRtm(m, rtMonitorSensGauge, jsonMetrics, sysMon)

	require.Greater(t, m.Alloc, uint64(0))
	require.Greater(t, len(jsonMetrics.jsonMetricsSlice), 0)
}

func TestParseKernMetrics(t *testing.T) {
	t.Parallel()
	cpuCount, _ := cpu.Counts(true)
	m := &kernelMetrics{CPUutilization: make([]metrictypes.Gauge, cpuCount)}
	j := &jsonModelsMetrics{}

	// parse kernel metrics
	parseKernMetrics(m, j)

	require.Greater(t, len(m.CPUutilization), 0)
	require.Greater(t, len(j.jsonMetricsSlice), 0)
}

func TestWorker(t *testing.T) {
	testIP := "::1"
	var crypt cryptandsign.AsymmetricCrypt = cryptandsign.NewAsymmetricCryptRsa()
	t.Parallel()
	ctx := context.Background()
	ts := httptest.NewServer(http.HandlerFunc(testServerHTTPHandler))
	t.Cleanup(func() { ts.Close() })

	id := 1
	ch1 := make(chan string, 1)
	ch1 <- "Hello"
	ch2 := make(chan error, 1)
	timeout := time.Second
	keyenc := ""
	pubkeypath := ""
	defer close(ch1)
	defer close(ch2)

	client := resty.New().R()

	go worker(ctx, id, ch1, timeout, ts.URL, keyenc, pubkeypath, client, ch2, crypt, testIP)
	require.NoError(t, <-ch2)
}
