package metrictypes

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMetricTypes(t *testing.T) {
	// test type association
	var (
		_ Gauge   = 0.1
		_ Counter = 1
	)

	require.Equal(t, GaugeType, "gauge")
	require.Equal(t, CounterType, "counter")
}
