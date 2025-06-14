package usecases

import (
	"context"
	"maps"
	"strconv"
	"testing"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/stretchr/testify/require"
)

func BenchmarkMetricsUsecase(b *testing.B) {
	storage := mockStorage{}
	usecase := NewMetricsUsecase(&storage)

	metricsCount := 100
	staticGauge := make(map[entities.MetricName]entities.Gauge)
	for i := range metricsCount {
		staticGauge["gauge_"+entities.MetricName(strconv.Itoa(i))] = entities.Gauge(i)
	}
	staticCounter := make(map[entities.MetricName]entities.Counter)
	for i := range metricsCount {
		staticCounter["counter_"+entities.MetricName(strconv.Itoa(i))] = entities.Counter(i)
	}

	storage.GetMetricsByTypesFunc = func(ctx context.Context,
		gauge map[entities.MetricName]entities.Gauge,
		counter map[entities.MetricName]entities.Counter,
	) error {
		maps.Copy(gauge, staticGauge)
		maps.Copy(counter, staticCounter)
		return nil
	}

	for b.Loop() {
		nextMetric, err := usecase.DumpIterator(context.Background())
		require.NoError(b, err)
		counter := 0
		for {
			_, _, _, exists := nextMetric()
			if !exists {
				break
			}
			counter++
		}
		require.Equal(b, 2*metricsCount, counter)
	}
}
