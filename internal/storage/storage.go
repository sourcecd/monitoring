package storage

import (
	"errors"
	"fmt"
	"sync"

	"github.com/sourcecd/monitoring/internal/metrictypes"
)

type StoreMetrics interface {
	WriteGauge(name string, value metrictypes.Gauge) error
	WriteCounter(name string, value metrictypes.Counter) error
	GetGauge(name string) (metrictypes.Gauge, error)
	GetCounter(name string) (metrictypes.Counter, error)
	GetAllMetricsTxt() string
}

// inmemory
type MemStorage struct {
	mx      sync.RWMutex
	gauge   map[string]metrictypes.Gauge
	counter map[string]metrictypes.Counter
}

func (m *MemStorage) WriteGauge(name string, value metrictypes.Gauge) error {
	m.mx.Lock()
	m.gauge[name] = value
	defer m.mx.Unlock()
	return nil
}
func (m *MemStorage) WriteCounter(name string, value metrictypes.Counter) error {
	m.mx.Lock()
	m.counter[name] += value
	defer m.mx.Unlock()
	return nil
}
func (m *MemStorage) GetGauge(name string) (metrictypes.Gauge, error) {
	m.mx.RLock()
	if v, ok := m.gauge[name]; ok {
		return v, nil
	}
	defer m.mx.RUnlock()
	return metrictypes.Gauge(0), errors.New("no gauge value")
}
func (m *MemStorage) GetCounter(name string) (metrictypes.Counter, error) {
	m.mx.RLock()
	if v, ok := m.counter[name]; ok {
		return v, nil
	}
	defer m.mx.RUnlock()
	return metrictypes.Counter(0), errors.New("no counter value")
}
func (m *MemStorage) GetAllMetricsTxt() string {
	s := "---Counters---\n"
	m.mx.RLock()
	for k, v := range m.counter {
		s += fmt.Sprintf("%v: %v\n", k, v)
	}
	s += "---Gauge---\n"
	for k, v := range m.gauge {
		s += fmt.Sprintf("%v: %v\n", k, v)
	}
	defer m.mx.RUnlock()
	return s
}
func (m *MemStorage) Setup() *MemStorage {
	m.gauge = make(map[string]metrictypes.Gauge)
	m.counter = make(map[string]metrictypes.Counter)

	return m
}
