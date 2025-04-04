package storage

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

const updateInterval = 100 * time.Millisecond

type FileStorage struct {
	mutex sync.RWMutex

	GaugeMap   map[entities.MetricName]entities.Gauge   `json:"gauge"`
	CounterMap map[entities.MetricName]entities.Counter `json:"counter"`

	StoreInterval   int    `json:"-"`
	FileStoragePath string `json:"-"`
}

func NewFileStorage(ctx context.Context, wg *sync.WaitGroup,
	storeInterval int, fileStoragePath string, restore bool,
) *FileStorage {
	result := &FileStorage{
		GaugeMap:        make(map[entities.MetricName]entities.Gauge),
		CounterMap:      make(map[entities.MetricName]entities.Counter),
		StoreInterval:   storeInterval,
		FileStoragePath: fileStoragePath,
	}

	if restore {
		result.loadMetrics()
	} else {
		slog.Info("[main] metrics file loading skipped", "path", fileStoragePath)
	}

	if storeInterval > 0 {
		wg.Add(1)
		storeInterval := time.Duration(storeInterval) * time.Second
		go func() {
			defer wg.Done()
			slog.Info("[preserver] start")
			for {
				result.storeMetrics("preserver")
				// sleep storeInterval or interrupt
				for t := time.Duration(0); t < storeInterval; t += updateInterval {
					select {
					case <-ctx.Done():
						// Handle context cancellation (graceful shutdown)
						slog.Info("[preserver] stopping", "error", ctx.Err())
						result.storeMetrics("preserver") // save changes
						return
					default:
						time.Sleep(updateInterval)
					}
				}
			}
		}()
	}

	return result
}

func (s *FileStorage) GetMetric(metric entities.Metric) (*entities.Metric, error) {
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
		"unexpected internal metric type: " + metric.Type.String())
}

func (s *FileStorage) UpdateMetric(metric entities.Metric) (*entities.Metric, error) {
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
		"unexpected internal metric type: " + metric.Type.String())
}

func (s *FileStorage) GetMetricsByTypes() (gauge map[entities.MetricName]entities.Gauge, counter map[entities.MetricName]entities.Counter) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	gauge = make(map[entities.MetricName]entities.Gauge)
	for k, v := range s.GaugeMap {
		gauge[k] = v
	}

	counter = make(map[entities.MetricName]entities.Counter)
	for k, v := range s.CounterMap {
		counter[k] = v
	}
	return
}

func (s *FileStorage) Ping() error { return nil }

func (s *FileStorage) Close() error { return nil }

func (s *FileStorage) loadMetrics() {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	file, err := os.Open(s.FileStoragePath)
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
	slog.Info("[main] metrics file loaded", "path", s.FileStoragePath)
}

func (s *FileStorage) storeMetrics(caller string) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	file, err := os.Create(s.FileStoragePath)
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
	slog.Info(msg, "path", s.FileStoragePath)
}

func (s *FileStorage) storeMetricsOnChangeIfRequired() {
	if s.StoreInterval <= 0 {
		s.storeMetrics("on change")
	}
}
