package usecases

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/PiskarevSA/go-advanced/internal/errors"
)

type Repositories interface {
	SetGauge(key string, value float64)
	Gauge(key string) (value float64, exists bool)
	IncreaseCounter(key string, delta int64) int64
	Counter(key string) (value int64, exists bool)
	Dump() (gauge map[string]float64, counter map[string]int64)
	Load(r io.Reader) error
	Store(w io.Writer) error
}

type DumpRow struct {
	type_ string
	name  string
	value string
}

type IteratableDump struct {
	rows  []DumpRow
	index int
}

func NewIteratableDump(gauge map[string]float64, counter map[string]int64) *IteratableDump {
	result := IteratableDump{}

	var keys []string
	for k := range gauge {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		result.rows = append(result.rows, DumpRow{"gauge", k, fmt.Sprint(gauge[k])})
	}

	keys = make([]string, 0)
	for k := range counter {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		result.rows = append(result.rows, DumpRow{"counter", k, fmt.Sprint(counter[k])})
	}

	return &result
}

func (d *IteratableDump) NextMetric() (
	type_ string, name string, value string, exists bool,
) {
	if d.index < len(d.rows) {
		row := &d.rows[d.index]
		d.index += 1
		return row.type_, row.name, row.value, true
	}
	return "", "", "", false
}

type Metrics struct {
	repo     Repositories
	OnChange func()
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
		m.repo.SetGauge(metricName, f64)
		if m.OnChange != nil {
			m.OnChange()
		}
	case "counter":
		if len(metricName) == 0 {
			return errors.NewEmptyMetricNameError()
		}
		i64, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return errors.NewMetricValueIsNotValidError(err)
		}
		m.repo.IncreaseCounter(metricName, i64)
		if m.OnChange != nil {
			m.OnChange()
		}
	case "":
		return errors.NewEmptyMetricTypeError()
	default:
		return errors.NewInvalidMetricTypeError(metricType)
	}
	return nil
}

func (m *Metrics) IsGauge(metricType string) (bool, error) {
	switch metricType {
	case "gauge":
		return true, nil
	case "counter":
		return false, nil
	case "":
		return false, errors.NewEmptyMetricTypeError()
	default:
		return false, errors.NewInvalidMetricTypeError(metricType)
	}
}

func (m *Metrics) UpdateGauge(metricName string, value *float64) error {
	if len(metricName) == 0 {
		return errors.NewEmptyMetricNameError()
	}
	if value == nil {
		return errors.NewMissingValueError()
	}
	m.repo.SetGauge(metricName, *value)
	if m.OnChange != nil {
		m.OnChange()
	}

	return nil
}

func (m *Metrics) IncreaseCounter(metricName string, delta *int64) (*int64, error) {
	if len(metricName) == 0 {
		return nil, errors.NewEmptyMetricNameError()
	}
	if delta == nil {
		return nil, errors.NewMissingDeltaError()
	}
	sum := m.repo.IncreaseCounter(metricName, *delta)
	if m.OnChange != nil {
		m.OnChange()
	}

	return &sum, nil
}

func (m *Metrics) Get(metricType string, metricName string) (
	value string, err error,
) {
	switch metricType {
	case "gauge":
		gauge, exists := m.repo.Gauge(metricName)
		if !exists {
			return "", errors.NewMetricNameNotFoundError(metricName)
		}
		return fmt.Sprint(gauge), nil

	case "counter":
		counter, exists := m.repo.Counter(metricName)
		if !exists {
			return "", errors.NewMetricNameNotFoundError(metricName)
		}
		return fmt.Sprint(counter), nil
	case "":
		return "", errors.NewEmptyMetricTypeError()
	default:
		return "", errors.NewInvalidMetricTypeError(metricType)
	}
}

func (m *Metrics) GetGauge(metricName string) (value *float64, err error) {
	gauge, exists := m.repo.Gauge(metricName)
	if !exists {
		return nil, errors.NewMetricNameNotFoundError(metricName)
	}
	return &gauge, nil
}

func (m *Metrics) GetCounter(metricName string) (value *int64, err error) {
	counter, exists := m.repo.Counter(metricName)
	if !exists {
		return nil, errors.NewMetricNameNotFoundError(metricName)
	}
	return &counter, nil
}

func (m *Metrics) DumpIterator() func() (type_ string, name string, value string, exists bool) {
	gauge, counter := m.repo.Dump()

	iteratableDump := NewIteratableDump(gauge, counter)
	return iteratableDump.NextMetric
}

func (m *Metrics) LoadMetrics(r io.Reader) error {
	if err := m.repo.Load(r); err != nil {
		return fmt.Errorf("load metrics: %w", err)
	}
	return nil
}

func (m *Metrics) StoreMetrics(w io.Writer) error {
	if err := m.repo.Store(w); err != nil {
		return fmt.Errorf("store metrics: %w", err)
	}
	return nil
}
