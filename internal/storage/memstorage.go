package storage

import (
	"context"
	"fmt"
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

func (s *MemStorage) GetMetric(ctx context.Context, metric entities.Metric,
) (*entities.Metric, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	switch metric.Type {
	case entities.MetricTypeGauge:
		value, exists := s.GaugeMap[metric.Name]
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
		delta, exists := s.CounterMap[metric.Name]
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
		"unexpected internal metric type: "+metric.Type.String(), nil)
}

func (s *MemStorage) UpdateMetric(ctx context.Context, metric entities.Metric,
) (*entities.Metric, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch metric.Type {
	case entities.MetricTypeGauge:
		s.GaugeMap[metric.Name] = metric.Value

		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: s.GaugeMap[metric.Name],
			Delta: 0,
		}
		return &result, nil
	case entities.MetricTypeCounter:
		s.CounterMap[metric.Name] += metric.Delta

		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: 0,
			Delta: s.CounterMap[metric.Name],
		}
		return &result, nil
	}
	return nil, entities.NewInternalError(
		"unexpected internal metric type: "+metric.Type.String(), nil)
}

func (s *MemStorage) UpdateMetrics(ctx context.Context, metrics []entities.Metric,
) ([]entities.Metric, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	NewGaugeMap := make(map[entities.MetricName]entities.Gauge)
	for k, v := range s.GaugeMap {
		NewGaugeMap[k] = v
	}
	NewCounterMap := make(map[entities.MetricName]entities.Counter)
	for k, v := range s.CounterMap {
		NewCounterMap[k] = v
	}

	result := make([]entities.Metric, 0)

	for i, metric := range metrics {
		switch metric.Type {
		case entities.MetricTypeGauge:
			NewGaugeMap[metric.Name] = metric.Value

			entityMetric := entities.Metric{
				Type:  metric.Type,
				Name:  metric.Name,
				Value: NewGaugeMap[metric.Name],
				Delta: 0,
			}
			result = append(result, entityMetric)
		case entities.MetricTypeCounter:
			NewCounterMap[metric.Name] += metric.Delta

			entityMetric := entities.Metric{
				Type:  metric.Type,
				Name:  metric.Name,
				Value: 0,
				Delta: NewCounterMap[metric.Name],
			}
			result = append(result, entityMetric)
		default:
			return nil, entities.NewInternalError(fmt.Sprintf(
				"metric[%v]: unexpected internal metric type: %v",
				i, metric.Type.String()), nil)
		}
	}

	s.GaugeMap = NewGaugeMap
	s.CounterMap = NewCounterMap
	return result, nil
}

func (s *MemStorage) GetMetricsByTypes(ctx context.Context,
	gauge map[entities.MetricName]entities.Gauge,
	counter map[entities.MetricName]entities.Counter,
) error {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	for k, v := range s.GaugeMap {
		gauge[k] = v
	}

	for k, v := range s.CounterMap {
		counter[k] = v
	}
	return nil
}

func (s *MemStorage) Ping(ctx context.Context) error { return nil }

func (s *MemStorage) Close(ctx context.Context) error { return nil }
