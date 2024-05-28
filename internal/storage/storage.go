package storage

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"

	"github.com/sourcecd/monitoring/internal/customerrors"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
)

type StoreMetrics interface {
	WriteMetric(ctx context.Context, mType, name string, val interface{}) error
	WriteBatchMetrics(ctx context.Context, metrics []models.Metrics) error
	GetAllMetricsTxt(ctx context.Context) (string, error)
	GetMetric(ctx context.Context, mType, name string) (interface{}, error)
	Ping(ctx context.Context) error
}

// inmemory
type MemStorage struct {
	mx      sync.RWMutex
	gauge   map[string]metrictypes.Gauge
	counter map[string]metrictypes.Counter
}

// TODO remove
func (m *MemStorage) Ping(ctx context.Context) error {
	return nil
}

func (m *MemStorage) WriteMetric(ctx context.Context, mtype, name string, val interface{}) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	switch mtype {
	case metrictypes.GaugeType:
		if metric, ok := val.(metrictypes.Gauge); ok {
			m.gauge[name] = metric
			return nil
		}
		return customerrors.ErrWrongMetricValueType
	case metrictypes.CounterType:
		if metric, ok := val.(metrictypes.Counter); ok {
			m.counter[name] += metric
			return nil
		}
		return customerrors.ErrWrongMetricValueType
	default:
		return customerrors.ErrWrongMetricType
	}
}
func (m *MemStorage) WriteBatchMetrics(ctx context.Context, metrics []models.Metrics) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	// i think we don't break all batch if one metric failed in batch (use continue)
	for _, v := range metrics {
		switch v.MType {
		case metrictypes.GaugeType:
			if v.Value == nil || v.ID == "" {
				log.Println("empty id or nil value gauge metric")
				continue
			}
			m.gauge[v.ID] = metrictypes.Gauge(*v.Value)
		case metrictypes.CounterType:
			if v.Delta == nil || v.ID == "" {
				log.Println("empty id or nil value counter metric")
				continue
			}
			m.counter[v.ID] += metrictypes.Counter(*v.Delta)
		default:
			log.Println("wrong metric type")
			continue
		}
	}
	return nil
}
func (m *MemStorage) GetMetric(ctx context.Context, mType, name string) (interface{}, error) {
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
		return nil, customerrors.ErrBadMetricType
	}
	return nil, customerrors.ErrNoVal
}
func (m *MemStorage) GetAllMetricsTxt(ctx context.Context) (string, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	s := "---Counters---\n"
	ck := make([]string, 0, len(m.counter))
	for k := range m.counter {
		ck = append(ck, k)
	}
	sort.Strings(ck)
	for _, k := range ck {
		s += fmt.Sprintf("%v: %v\n", k, m.counter[k])
	}
	s += "---Gauge---\n"
	gk := make([]string, 0, len(m.gauge))
	for k := range m.gauge {
		gk = append(gk, k)
	}
	sort.Strings(gk)
	for _, k := range gk {
		s += fmt.Sprintf("%v: %v\n", k, m.gauge[k])
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
