package storage

import (
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

func (m *MemStorage) GetMetric(metric entities.Metric) (*entities.Metric, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	switch metric.Type {
	case entities.MetricTypeGauge:
		value, exists := m.GaugeMap[metric.Name]
		if !exists {
			return nil, entities.NewMetricNameNotFoundError(metric.Name)
		}
		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: value,
			Delta: 0,
		}
		return &result, nil
	case entities.MetricTypeCounter:
		delta, exists := m.CounterMap[metric.Name]
		if !exists {
			return nil, entities.NewMetricNameNotFoundError(metric.Name)
		}
		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: 0,
			Delta: delta,
		}
		return &result, nil
	}
	return nil, entities.NewInternalError(
		"unexpected internal metric type: " + metric.Type.String())
}

func (m *MemStorage) UpdateMetric(metric entities.Metric) (*entities.Metric, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	switch metric.Type {
	case entities.MetricTypeGauge:
		m.GaugeMap[metric.Name] = metric.Value

		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: m.GaugeMap[metric.Name],
			Delta: 0,
		}
		return &result, nil
	case entities.MetricTypeCounter:
		m.CounterMap[metric.Name] += metric.Delta

		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: 0,
			Delta: m.CounterMap[metric.Name],
		}
		return &result, nil
	}
	return nil, entities.NewInternalError(
		"unexpected internal metric type: " + metric.Type.String())
}

func (m *MemStorage) GetMetricsByTypes() (gauge map[entities.MetricName]entities.Gauge, counter map[entities.MetricName]entities.Counter) {
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

func (m *MemStorage) Ping() error { return nil }

func (m *MemStorage) Close() error { return nil }
