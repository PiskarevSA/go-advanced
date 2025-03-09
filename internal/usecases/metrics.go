package usecases

import (
	"fmt"
	"sort"
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
		m.repo.SetGauge(metricName, f64)
	case "counter":
		if len(metricName) == 0 {
			return errors.NewEmptyMetricNameError()
		}
		i64, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return errors.NewMetricValueIsNotValidError(err)
		}
		m.repo.SetCounter(metricName, i64)
	case "":
		return errors.NewEmptyMetricTypeError()
	default:
		return errors.NewInvalidMetricTypeError(metricType)
	}
	return nil
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

func (m *Metrics) DumpIterator() func() (type_ string, name string, value string, exists bool) {
	gauge, counter := m.repo.Dump()

	iteratableDump := NewIteratableDump(gauge, counter)
	return iteratableDump.NextMetric
}
