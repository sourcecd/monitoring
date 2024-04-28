package agent

import (
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sourcecd/monitoring/internal/metrictypes"
)

func TestMetricsAgent(t *testing.T) {
	rtm := &runtime.MemStats{}
	sysMetrics := &sysMon{}
	numCount := 1
	randVal := 0
	testAllocSens := 0
	testrandVal := metrictypes.Gauge(0.123)
	testPollCount := metrictypes.Counter(5)
	testRtFiledName := "Alloc"
	expSysMetricURLs := &sysMetricsJSON{
		metricRandJ: `{"id":"RandomValue","type":"gauge","value":0.123}`,
		metricPollCountJ: `{"id":"PollCount","type":"counter","delta":5}`,
	}
	expRtMetricURL := `{"id":"Alloc","type":"gauge","value":0}`
	// get val
	m := reflect.ValueOf(rtm).Elem()
	stringTestRt := fmt.Sprintf("%v", m.FieldByName(testRtFiledName))
	fl64, err := strconv.ParseFloat(fmt.Sprintf("%v", stringTestRt), 64)
	assert.NoError(t, err)

	parsedRtMetricURLres := parsedRtMetricURL(testRtFiledName, fl64)

	updateMetrics(rtm, sysMetrics)
	urls := parsedSysMetricsURL(testrandVal, testPollCount)

	assert.Equal(t, metrictypes.Counter(numCount), sysMetrics.pollCount)
	assert.NotEqual(t, metrictypes.Gauge(randVal), sysMetrics.randomValue)
	assert.NotEqual(t, uint64(testAllocSens), rtm.Alloc)

	assert.Equal(t, expSysMetricURLs, urls)
	assert.Equal(t, expRtMetricURL, parsedRtMetricURLres)
}
