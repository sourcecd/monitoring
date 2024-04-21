package metrictypes

const (
	GaugeType = "gauge"
	CounterType = "counter"
)

type (
	Gauge   float64
	Counter int64
)
