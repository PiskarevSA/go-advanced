package storage

import (
	"encoding/json"
	"io"
	"sync"

	"github.com/PiskarevSA/go-advanced/internal/entities"
)

type MemStorage struct {
	mutex sync.RWMutex

	GaugeMap   map[entities.MetricName]entities.Gauge   `json:"gauge"`
	CounterMap map[entities.MetricName]entities.Counter `json:"counter"`
}

func NewMemStorage() *MemStorage {
	return &MemStorage{
		GaugeMap:   make(map[entities.MetricName]entities.Gauge),
		CounterMap: make(map[entities.MetricName]entities.Counter),
	}
}

func (m *MemStorage) Get(metric entities.Metric) (*entities.Metric, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if metric.IsGauge {
		value, exists := m.GaugeMap[metric.Name]
		if !exists {
			return nil, entities.NewMetricNameNotFoundError(metric.Name)
		}
		result := entities.Metric{
			IsGauge: true,
			Name:    metric.Name,
			Value:   value,
			Delta:   0,
		}
		return &result, nil
	} else {
		delta, exists := m.CounterMap[metric.Name]
		if !exists {
			return nil, entities.NewMetricNameNotFoundError(metric.Name)
		}
		result := entities.Metric{
			IsGauge: false,
			Name:    metric.Name,
			Value:   0,
			Delta:   delta,
		}
		return &result, nil
	}
}

func (m *MemStorage) Update(metric entities.Metric) (*entities.Metric, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if metric.IsGauge {
		m.GaugeMap[metric.Name] = metric.Value

		result := entities.Metric{
			IsGauge: true,
			Name:    metric.Name,
			Value:   m.GaugeMap[metric.Name],
			Delta:   0,
		}
		return &result, nil
	} else {
		m.CounterMap[metric.Name] += metric.Delta

		result := entities.Metric{
			IsGauge: false,
			Name:    metric.Name,
			Value:   0,
			Delta:   m.CounterMap[metric.Name],
		}
		return &result, nil
	}
}

func (m *MemStorage) Dump() (gauge map[entities.MetricName]entities.Gauge, counter map[entities.MetricName]entities.Counter) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	gauge = make(map[entities.MetricName]entities.Gauge)
	for k, v := range m.GaugeMap {
		gauge[k] = v
	}

	counter = make(map[entities.MetricName]entities.Counter)
	for k, v := range m.CounterMap {
		counter[k] = v
	}
	return
}

func (m *MemStorage) Load(r io.Reader) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return json.NewDecoder(r).Decode(&m)
}

func (m *MemStorage) Store(w io.Writer) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return json.NewEncoder(w).Encode(m)
}
