package usecases

import (
	"fmt"
	"io"
	"sort"

	"github.com/PiskarevSA/go-advanced/internal/entities"
)

type storage interface {
	GetMetric(metric entities.Metric) (*entities.Metric, error)
	UpdateMetric(metric entities.Metric) (*entities.Metric, error)
	GetMetricsByTypes() (gauge map[entities.MetricName]entities.Gauge,
		counter map[entities.MetricName]entities.Counter)
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

func NewIteratableDump(gauge map[entities.MetricName]entities.Gauge, counter map[entities.MetricName]entities.Counter) *IteratableDump {
	result := IteratableDump{}

	var keys []string
	for k := range gauge {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	for _, k := range keys {
		result.rows = append(result.rows,
			DumpRow{"gauge", k, fmt.Sprint(gauge[entities.MetricName(k)])})
	}

	keys = make([]string, 0)
	for k := range counter {
		keys = append(keys, string(k))
	}
	sort.Strings(keys)

	for _, k := range keys {
		result.rows = append(result.rows,
			DumpRow{"counter", k, fmt.Sprint(counter[entities.MetricName(k)])})
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

// MetricsUsecase contains use cases, related to metrics creating, reading and updating
type MetricsUsecase struct {
	storage  storage
	OnChange func()
}

func NewMetricsUsecase(storage storage) *MetricsUsecase {
	return &MetricsUsecase{
		storage: storage,
	}
}

func (m *MetricsUsecase) GetMetric(metric entities.Metric) (*entities.Metric, error) {
	return m.storage.GetMetric(metric)
}

func (m *MetricsUsecase) UpdateMetric(metric entities.Metric) (*entities.Metric, error) {
	return m.storage.UpdateMetric(metric)
}

func (m *MetricsUsecase) DumpIterator() func() (type_ string, name string, value string, exists bool) {
	gauge, counter := m.storage.GetMetricsByTypes()

	iteratableDump := NewIteratableDump(gauge, counter)
	return iteratableDump.NextMetric
}

func (m *MetricsUsecase) LoadMetrics(r io.Reader) error {
	if err := m.storage.Load(r); err != nil {
		return fmt.Errorf("load metrics: %w", err)
	}
	return nil
}

func (m *MetricsUsecase) StoreMetrics(w io.Writer) error {
	if err := m.storage.Store(w); err != nil {
		return fmt.Errorf("store metrics: %w", err)
	}
	return nil
}
