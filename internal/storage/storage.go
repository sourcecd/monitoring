package storage

import (
	"errors"

	"github.com/sourcecd/monitoring/internal/metrictypes"
)

type StoreMetrics interface {
	WriteGauge(name string, value metrictypes.Gauge) error
	WriteCounter(name string, value metrictypes.Counter) error
}

//inmemory
type MemStorage struct {
	gauge   map[string]metrictypes.Gauge
	counter map[string]metrictypes.Counter
}
func (m *MemStorage) WriteGauge(name string, value metrictypes.Gauge) error {
	m.gauge[name] = value
	return nil
}
func (m *MemStorage) WriteCounter(name string, value metrictypes.Counter) error {
	m.counter[name] += value
	return nil
}
func (m *MemStorage) GetGauge(name string) (metrictypes.Gauge, error) {
	if v, ok := m.gauge[name]; ok {
		return v, nil
	}
	return metrictypes.Gauge(0), errors.New("no gauge value")
}
func (m *MemStorage) GetCounter(name string) (metrictypes.Counter, error) {
	if v, ok := m.counter[name]; ok {
		return v, nil
	}
	return metrictypes.Counter(0), errors.New("no counter value")
}
func (m *MemStorage) Setup() *MemStorage {
	m.gauge = make(map[string]metrictypes.Gauge)
	m.counter = make(map[string]metrictypes.Counter)

	return m
}
