package usecases

import (
	"fmt"
	"strconv"

	"github.com/PiskarevSA/go-advanced/internal/errors"
)

type Repositories interface {
	SetGauge(key string, value float64)
	Gauge(key string) (value float64, exist bool)
	SetCounter(key string, value int64)
	Counter(key string) (value int64, exist bool)
	Dump() (gauge map[string]float64, counter map[string]int64)
}

type Metrics struct {
	repo Repositories
}

func NewMetrics(repo Repositories) *Metrics {
	return &Metrics{
		repo: repo,
	}
}

func (m *Metrics) Update(metricType string, metricName string, metricValue string) error {
	switch metricType {
	case "gauge":
		if len(metricName) == 0 {
			return errors.NewEmptyMetricNameError()
		}
		f64, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return errors.NewMetricValueIsNotValidError(err)
		}
		m.SetGauge(metricName, f64)
	case "counter":
		if len(metricName) == 0 {
			return errors.NewEmptyMetricNameError()
		}
		i64, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return errors.NewMetricValueIsNotValidError(err)
		}
		m.SetCounter(metricName, i64)
	case "":
		return errors.NewEmptyMetricTypeError()
	default:
		return errors.NewInvalidMetricTypeError(metricType)
	}
	return nil
}

func (m *Metrics) SetGauge(key string, value float64) {
	m.repo.SetGauge(key, value)
}

func (m *Metrics) Get(metricType string, metricName string) (
	value string, err error,
) {
	switch metricType {
	case "gauge":
		gauge, exist := m.repo.Gauge(metricName)
		if !exist {
			return "", errors.NewMetricNameNotFoundError(metricName)
		}
		return fmt.Sprint(gauge), nil

	case "counter":
		counter, exist := m.repo.Counter(metricName)
		if !exist {
			return "", errors.NewMetricNameNotFoundError(metricName)
		}
		return fmt.Sprint(counter), nil
	case "":
		return "", errors.NewEmptyMetricTypeError()
	default:
		return "", errors.NewInvalidMetricTypeError(metricType)
	}
}

func (m *Metrics) SetCounter(key string, value int64) {
	m.repo.SetCounter(key, value)
}

func (m *Metrics) Dump() (gauge map[string]float64, counter map[string]int64) {
	return m.repo.Dump()
}
