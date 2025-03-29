package usecases

import (
	"fmt"
	"io"
	"sort"
	"strconv"

	"github.com/PiskarevSA/go-advanced/internal/entities"
)

type Storage interface {
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

// Metrics contains use cases, related to metrics creating, reading and updating
type Metrics struct {
	storage  Storage
	OnChange func()
}

func NewMetrics(storage Storage) *Metrics {
	return &Metrics{
		storage: storage,
	}
}

func (m *Metrics) UpdateMetric(type_ string, name string, value string) error {
	switch type_ {
	case "gauge":
		if len(name) == 0 {
			return entities.ErrEmptyMetricName
		}
		asFloat64, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return entities.NewMetricValueIsNotValidError(err)
		}
		m.storage.SetGauge(name, asFloat64)
		if m.OnChange != nil {
			m.OnChange()
		}
	case "counter":
		if len(name) == 0 {
			return entities.ErrEmptyMetricName
		}
		asInt64, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return entities.NewMetricValueIsNotValidError(err)
		}
		m.storage.IncreaseCounter(name, asInt64)
		if m.OnChange != nil {
			m.OnChange()
		}
	default:
		return entities.NewInvalidMetricTypeError(type_)
	}
	return nil
}

func (m *Metrics) IsGauge(type_ string) (bool, error) {
	switch type_ {
	case "gauge":
		return true, nil
	case "counter":
		return false, nil
	default:
		return false, entities.NewInvalidMetricTypeError(type_)
	}
}

func (m *Metrics) UpdateGauge(name string, value *float64) error {
	if len(name) == 0 {
		return entities.ErrEmptyMetricName
	}
	if value == nil {
		return entities.ErrMissingValue
	}
	m.storage.SetGauge(name, *value)
	if m.OnChange != nil {
		m.OnChange()
	}

	return nil
}

func (m *Metrics) IncreaseCounter(name string, delta *int64) (*int64, error) {
	if len(name) == 0 {
		return nil, entities.ErrEmptyMetricName
	}
	if delta == nil {
		return nil, entities.ErrMissingDelta
	}
	sum := m.storage.IncreaseCounter(name, *delta)
	if m.OnChange != nil {
		m.OnChange()
	}

	return &sum, nil
}

func (m *Metrics) GetMetric(type_ string, name string) (
	value string, err error,
) {
	switch type_ {
	case "gauge":
		gauge, exists := m.storage.Gauge(name)
		if !exists {
			return "", entities.NewMetricNameNotFoundError(name)
		}
		return fmt.Sprint(gauge), nil

	case "counter":
		counter, exists := m.storage.Counter(name)
		if !exists {
			return "", entities.NewMetricNameNotFoundError(name)
		}
		return fmt.Sprint(counter), nil
	default:
		return "", entities.NewInvalidMetricTypeError(type_)
	}
}

func (m *Metrics) GetGauge(name string) (value *float64, err error) {
	gauge, exists := m.storage.Gauge(name)
	if !exists {
		return nil, entities.NewMetricNameNotFoundError(name)
	}
	return &gauge, nil
}

func (m *Metrics) GetCounter(name string) (value *int64, err error) {
	counter, exists := m.storage.Counter(name)
	if !exists {
		return nil, entities.NewMetricNameNotFoundError(name)
	}
	return &counter, nil
}

func (m *Metrics) DumpIterator() func() (type_ string, name string, value string, exists bool) {
	gauge, counter := m.storage.Dump()

	iteratableDump := NewIteratableDump(gauge, counter)
	return iteratableDump.NextMetric
}

func (m *Metrics) LoadMetrics(r io.Reader) error {
	if err := m.storage.Load(r); err != nil {
		return fmt.Errorf("load metrics: %w", err)
	}
	return nil
}

func (m *Metrics) StoreMetrics(w io.Writer) error {
	if err := m.storage.Store(w); err != nil {
		return fmt.Errorf("store metrics: %w", err)
	}
	return nil
}
