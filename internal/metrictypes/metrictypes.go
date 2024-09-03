// Package metrictypes with metric types.
package metrictypes

import (
	"context"
	"sync"

	"github.com/sourcecd/monitoring/internal/models"
)

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

// JSONModelsMetrics type of collection gauge and counter metrics.
type JSONModelsMetrics struct {
	JSONMetricsSlice []models.Metrics
	sync.RWMutex
}

type MetricSender interface {
	Send(ctx context.Context, serverHost, xRealIp string) error
}
