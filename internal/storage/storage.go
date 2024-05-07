package storage

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
)

type StoreMetrics interface {
	WriteMetric(mType, name string, val interface{}) error
	GetAllMetricsTxt() (string, error)
	GetMetric(mType, name string) (interface{}, error)
	Ping() error
}

// inmemory
type MemStorage struct {
	mx      sync.RWMutex
	gauge   map[string]metrictypes.Gauge
	counter map[string]metrictypes.Counter
}

// TODO remove
func (m *MemStorage) Ping() error {
	return nil
}

func (m *MemStorage) WriteMetric(mtype, name string, val interface{}) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if mtype == metrictypes.GaugeType {
		if metric, ok := val.(metrictypes.Gauge); ok {
			m.gauge[name] = metric
			return nil
		}
		return errors.New("wrong metric value type")
	} else if mtype == metrictypes.CounterType {
		if metric, ok := val.(metrictypes.Counter); ok {
			m.counter[name] += metric
			return nil
		}
		return errors.New("wrong metric value type")
	}
	return errors.New("wrong metric type")
}
func (m *MemStorage) GetMetric(mType, name string) (interface{}, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	if mType == metrictypes.GaugeType {
		if v, ok := m.gauge[name]; ok {
			return v, nil
		}
	} else if mType == metrictypes.CounterType {
		if v, ok := m.counter[name]; ok {
			return v, nil
		}
	} else {
		return nil, errors.New("bad metric type")
	}
	return nil, errors.New("no value")
}
func (m *MemStorage) GetAllMetricsTxt() (string, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	s := "---Counters---\n"
	for k, v := range m.counter {
		s += fmt.Sprintf("%v: %v\n", k, v)
	}
	s += "---Gauge---\n"
	for k, v := range m.gauge {
		s += fmt.Sprintf("%v: %v\n", k, v)
	}
	return s, nil
}

func (m *MemStorage) SaveToFile(fname string) error {
	f, err := os.Create(fname)
	if err != nil {
		return err
	}

	m.mx.RLock()
	defer m.mx.RUnlock()

	defer func() {
		_ = f.Close()
	}()

	enc := json.NewEncoder(f)

	for k, v := range m.counter {
		_ = enc.Encode(models.Metrics{
			MType: metrictypes.CounterType,
			ID:    k,
			Delta: (*int64)(&v),
		})
	}
	for k, v := range m.gauge {
		_ = enc.Encode(models.Metrics{
			MType: metrictypes.GaugeType,
			ID:    k,
			Value: (*float64)(&v),
		})
	}
	return nil
}

func (m *MemStorage) ReadFromFile(fname string) error {
	f, err := os.Open(fname)
	if err != nil {
		return err
	}

	m.mx.Lock()
	defer m.mx.Unlock()

	defer func() {
		_ = f.Close()
	}()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		metric := &models.Metrics{}
		if err := json.Unmarshal(scanner.Bytes(), metric); err != nil {
			return err
		}
		switch metric.MType {
		case metrictypes.CounterType:
			m.counter[metric.ID] = metrictypes.Counter(*metric.Delta)
		case metrictypes.GaugeType:
			m.gauge[metric.ID] = metrictypes.Gauge(*metric.Value)
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return nil
}

func NewMemStorage() *MemStorage {
	return &MemStorage{gauge: make(map[string]metrictypes.Gauge), counter: make(map[string]metrictypes.Counter)}
}
