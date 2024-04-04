package agent

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sourcecd/monitoring/internal/metrictypes"
)

func TestMetricsAgent(t *testing.T) {
	rtm := &runtime.MemStats{}
	sysMetrics := &SysMon{}
	numCount := 1
	randVal := 0
	testAllocSens := 0
	testHost := "http://localhost:8080"
	testrandVal := metrictypes.Gauge(0.123)
	testPollCount := metrictypes.Counter(5)
	testRtFiledName := "Alloc"
	expSysMetricURLs := []string{
		"http://localhost:8080/update/gauge/randomvalue/0.123000",
		"http://localhost:8080/update/counter/pollcount/5",
	}
	expRtMetricURL := "http://localhost:8080/update/gauge/alloc/0"
	m := reflect.ValueOf(rtm).Elem()

	parsedRtMetricURLres := parsedRtMetricURL(testHost, testRtFiledName, m.FieldByName(testRtFiledName))

	updateMetrics(rtm, sysMetrics)
	urls := parsedSysMetricsURL(testHost, testrandVal, testPollCount)

	assert.Equal(t, metrictypes.Counter(numCount), sysMetrics.PollCount)
	assert.NotEqual(t, metrictypes.Gauge(randVal), sysMetrics.RandomValue)
	assert.NotEqual(t, uint64(testAllocSens), rtm.Alloc)

	assert.Equal(t, expSysMetricURLs, urls)
	assert.Equal(t, expRtMetricURL, parsedRtMetricURLres)
}
