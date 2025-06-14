package usecases

import (
	"context"
	"fmt"
	"sort"

	"github.com/PiskarevSA/go-advanced/internal/entities"
)

type storage interface {
	GetMetric(ctx context.Context, metric entities.Metric) (*entities.Metric, error)
	UpdateMetric(ctx context.Context, metric entities.Metric) (*entities.Metric, error)
	UpdateMetrics(ctx context.Context, metrics []entities.Metric) ([]entities.Metric, error)
	GetMetricsByTypes(ctx context.Context, gauge map[entities.MetricName]entities.Gauge,
		counter map[entities.MetricName]entities.Counter) error
	Ping(ctx context.Context) error
	Close(ctx context.Context) error
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

func NewIteratableDump(gauge map[entities.MetricName]entities.Gauge,
	counter map[entities.MetricName]entities.Counter,
) *IteratableDump {
	gaugeKeys := make([]string, 0, len(gauge))
	for k := range gauge {
		gaugeKeys = append(gaugeKeys, string(k))
	}
	sort.Strings(gaugeKeys)

	counterKeys := make([]string, 0, len(counter))
	for k := range counter {
		counterKeys = append(counterKeys, string(k))
	}
	sort.Strings(counterKeys)

	result := IteratableDump{
		rows: make([]DumpRow, 0, len(gaugeKeys)+len(counterKeys)),
	}

	for _, k := range gaugeKeys {
		result.rows = append(result.rows,
			DumpRow{"gauge", k, fmt.Sprint(gauge[entities.MetricName(k)])})
	}

	for _, k := range counterKeys {
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
	storage storage
}

func NewMetricsUsecase(storage storage) *MetricsUsecase {
	return &MetricsUsecase{
		storage: storage,
	}
}

func (m *MetricsUsecase) GetMetric(ctx context.Context, metric entities.Metric,
) (*entities.Metric, error) {
	return m.storage.GetMetric(ctx, metric)
}

func (m *MetricsUsecase) UpdateMetric(ctx context.Context, metric entities.Metric,
) (*entities.Metric, error) {
	return m.storage.UpdateMetric(ctx, metric)
}

func (m *MetricsUsecase) UpdateMetrics(ctx context.Context, metrics []entities.Metric,
) ([]entities.Metric, error) {
	return m.storage.UpdateMetrics(ctx, metrics)
}

func (m *MetricsUsecase) DumpIterator(ctx context.Context) (func() (type_ string, name string, value string, exists bool), error) {
	gauge := make(map[entities.MetricName]entities.Gauge)
	counter := make(map[entities.MetricName]entities.Counter)
	if err := m.storage.GetMetricsByTypes(ctx, gauge, counter); err != nil {
		return nil, err
	}

	iteratableDump := NewIteratableDump(gauge, counter)
	return iteratableDump.NextMetric, nil
}

func (m *MetricsUsecase) Ping(ctx context.Context) error {
	return m.storage.Ping(ctx)
}
