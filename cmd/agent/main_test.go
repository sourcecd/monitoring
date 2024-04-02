package main

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMetricsAgent(t *testing.T) {
	rtm := &runtime.MemStats{}
	sysMetrics := &SysMon{}
	pollInterval := 0
	numCount := 1
	randVal := 0
	testAllocSens := 0
	testHost := "http://localhost:8080"
	testrandVal := gauge(0.123)
	testPollCount := counter(5)
	testRtFiledName := "Alloc"
	expSysMetricURLs := []string{
		"http://localhost:8080/update/gauge/randomvalue/0.123000",
		"http://localhost:8080/update/counter/pollcount/5",
	}
	expRtMetricURL := "http://localhost:8080/update/gauge/alloc/0"
	m := reflect.ValueOf(rtm).Elem()

	parsedRtMetricURLres := parsedRtMetricURL(testHost, testRtFiledName, m.FieldByName(testRtFiledName))

	updateMetrics(rtm, sysMetrics, pollInterval)
	urls := parsedSysMetricsURL(testHost, testrandVal, testPollCount)

	assert.Equal(t, counter(numCount), sysMetrics.PollCount)
	assert.NotEqual(t, gauge(randVal), sysMetrics.RandomValue)
	assert.NotEqual(t, uint64(testAllocSens), rtm.Alloc)

	assert.Equal(t, expSysMetricURLs, urls)
	assert.Equal(t, expRtMetricURL, parsedRtMetricURLres)
}
