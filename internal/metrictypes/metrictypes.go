// Package with metric types.
package metrictypes

const (
	// Gauge.
	GaugeType = "gauge"
	// Counter.
	CounterType = "counter"
)

type (
	// Base type for gauge metric - float64
	Gauge float64
	// Base type for counter metric - int64
	Counter int64
)
