package filestorage

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/PiskarevSA/go-advanced/internal/entities"
)

type FileStorage struct {
	mutex sync.RWMutex

	GaugeMap   map[entities.MetricName]entities.Gauge   `json:"gauge"`
	CounterMap map[entities.MetricName]entities.Counter `json:"counter"`

	storeInterval   int
	fileStoragePath string
}

func New(ctx context.Context, wg *sync.WaitGroup,
	storeInterval int, fileStoragePath string, restore bool,
) *FileStorage {
	result := &FileStorage{
		GaugeMap:        make(map[entities.MetricName]entities.Gauge),
		CounterMap:      make(map[entities.MetricName]entities.Counter),
		storeInterval:   storeInterval,
		fileStoragePath: fileStoragePath,
	}

	if restore {
		result.loadMetrics()
	} else {
		slog.Info("[main] metrics file loading skipped", "path", fileStoragePath)
	}

	if storeInterval > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			slog.Info("[preserver] start")

			store := func() {
				result.storeMetrics("preserver")
			}

			ticker := time.NewTicker(time.Duration(storeInterval) * time.Second)
			stop := func() {
				slog.Info("[preserver] stopping", "error", ctx.Err())
				ticker.Stop()
			}

			// make first store instantly
			select {
			case <-ctx.Done():
				stop()
				store() // store on exit
				return
			default:
				store()
			}

			// use ticker after that
			for {
				select {
				case <-ctx.Done():
					stop()
					return
				case <-ticker.C:
					store()
				}
			}
		}()
	}

	return result
}

func (s *FileStorage) GetMetric(ctx context.Context, metric entities.Metric,
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

func (s *FileStorage) UpdateMetric(ctx context.Context, metric entities.Metric,
) (*entities.Metric, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	switch metric.Type {
	case entities.MetricTypeGauge:
		s.GaugeMap[metric.Name] = metric.Value
		s.storeMetricsOnChangeIfRequired()

		result := entities.Metric{
			Type:  metric.Type,
			Name:  metric.Name,
			Value: s.GaugeMap[metric.Name],
			Delta: 0,
		}
		return &result, nil
	case entities.MetricTypeCounter:
		s.CounterMap[metric.Name] += metric.Delta
		s.storeMetricsOnChangeIfRequired()

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

func (s *FileStorage) UpdateMetrics(ctx context.Context, metrics []entities.Metric,
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
	s.storeMetricsOnChangeIfRequired()
	return result, nil
}

func (s *FileStorage) GetMetricsByTypes(ctx context.Context,
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

func (s *FileStorage) Ping(ctx context.Context) error { return nil }

func (s *FileStorage) Close(ctx context.Context) error { return nil }

func (s *FileStorage) loadMetrics() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	file, err := os.Open(s.fileStoragePath)
	if err != nil {
		slog.Error("[main] open metrics file", "error", err.Error())
		return
	}
	defer file.Close()
	err = json.NewDecoder(file).Decode(&s)
	if err != nil {
		slog.Error("[main] load metrics file", "error", err.Error())
		return
	}
	slog.Info("[main] metrics file loaded", "path", s.fileStoragePath)
}

func (s *FileStorage) storeMetrics(caller string) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	file, err := os.Create(s.fileStoragePath)
	if err != nil {
		msg := fmt.Sprintf("[%v] create metrics file", caller)
		slog.Error(msg, "error", err.Error())
		return
	}
	defer file.Close()
	err = json.NewEncoder(file).Encode(s)
	if err != nil {
		msg := fmt.Sprintf("[%v] store metrics file", caller)
		slog.Error(msg, "error", err.Error())
		return
	}

	msg := fmt.Sprintf("[%v] metrics file stored", caller)
	slog.Info(msg, "path", s.fileStoragePath)
}

func (s *FileStorage) storeMetricsOnChangeIfRequired() {
	if s.storeInterval <= 0 {
		s.storeMetrics("on change")
	}
}
