// Package metrictypes with metric types.
package metrictypes

const (
	// Gauge.
	GaugeType = "gauge"
	// Counter.
	CounterType = "counter"
)

type (
	// Gauge base type for gauge metric - float64
	Gauge float64
	// Counter base type for counter metric - int64
	Counter int64
)
