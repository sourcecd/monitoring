package agent

import (
	"fmt"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
)

func TestMetricsAgent(t *testing.T) {
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
