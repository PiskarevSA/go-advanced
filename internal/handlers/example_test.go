package handlers_test

import (
	"context"
	"fmt"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/PiskarevSA/go-advanced/internal/storage/memstorage"
	"github.com/PiskarevSA/go-advanced/internal/usecases"
)

func Example() {
	storage := memstorage.New()
	usecase := usecases.NewMetricsUsecase(storage)
	_, err := usecase.UpdateMetric(context.Background(), entities.Metric{
		Type:  entities.MetricTypeCounter,
		Name:  entities.MetricName("foo"),
		Delta: 42,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	_, err = usecase.UpdateMetric(context.Background(), entities.Metric{
		Type:  entities.MetricTypeCounter,
		Name:  entities.MetricName("foo"),
		Delta: 100,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	metric, err := usecase.GetMetric(context.Background(), entities.Metric{
		Type: entities.MetricTypeCounter,
		Name: entities.MetricName("foo"),
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("foo counter:", metric.Delta)

	// Output:
	// foo counter: 142
}
