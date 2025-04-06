package storage

import (
	"context"
	"testing"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/stretchr/testify/assert"
)

func filledMemStorage() *MemStorage {
	return &MemStorage{
		GaugeMap: map[entities.MetricName]entities.Gauge{
			"Gauge1": 1.11,
			"Gauge2": 2.22,
		},
		CounterMap: map[entities.MetricName]entities.Counter{
			"Counter1": 111,
			"Counter2": 222,
		},
	}
}

func TestMemStorage_GetMetric(t *testing.T) {
	type given struct {
		argMetric entities.Metric
	}
	type want struct {
		argResponse *entities.Metric
		argError    error
	}
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "get existing gauge",
			given: given{
				argMetric: entities.Metric{
					Type: entities.MetricTypeGauge,
					Name: "Gauge2",
				},
			},
			want: want{
				argResponse: &entities.Metric{
					Type:  entities.MetricTypeGauge,
					Name:  "Gauge2",
					Value: 2.22,
				},
				argError: nil,
			},
		},
		{
			name: "get non-existing gauge",
			given: given{
				argMetric: entities.Metric{
					Type: entities.MetricTypeGauge,
					Name: "Gauge3",
				},
			},
			want: want{
				argResponse: nil,
				argError:    entities.NewMetricNameNotFoundError("Gauge3"),
			},
		},
		{
			name: "get existing counter",
			given: given{
				argMetric: entities.Metric{
					Type: entities.MetricTypeCounter,
					Name: "Counter2",
				},
			},
			want: want{
				argResponse: &entities.Metric{
					Type:  entities.MetricTypeCounter,
					Name:  "Counter2",
					Delta: 222,
				},
				argError: nil,
			},
		},
		{
			name: "get non-existing counter",
			given: given{
				argMetric: entities.Metric{
					Type: entities.MetricTypeCounter,
					Name: "Counter3",
				},
			},
			want: want{
				argResponse: nil,
				argError:    entities.NewMetricNameNotFoundError("Counter3"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := filledMemStorage().GetMetric(
				context.Background(), tt.given.argMetric)
			assert.Equal(t, tt.want.argResponse, response)
			assert.Equal(t, err, tt.want.argError)
		})
	}
}

func TestMemStorage_UpdateMetric(t *testing.T) {
	type given struct {
		argMetric entities.Metric
	}
	type want struct {
		argResponse *entities.Metric
		argError    error
	}
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "add gauge",
			given: given{
				argMetric: entities.Metric{
					Type:  entities.MetricTypeGauge,
					Name:  "Gauge3",
					Value: 3.33,
				},
			},
			want: want{
				argResponse: &entities.Metric{
					Type:  entities.MetricTypeGauge,
					Name:  "Gauge3",
					Value: 3.33,
				},
				argError: nil,
			},
		},
		{
			name: "replace gauge",
			given: given{
				argMetric: entities.Metric{
					Type:  entities.MetricTypeGauge,
					Name:  "Gauge2",
					Value: 22.22,
				},
			},
			want: want{
				argResponse: &entities.Metric{
					Type:  entities.MetricTypeGauge,
					Name:  "Gauge2",
					Value: 22.22,
				},
				argError: nil,
			},
		},
		{
			name: "add counter",
			given: given{
				argMetric: entities.Metric{
					Type:  entities.MetricTypeCounter,
					Name:  "Counter3",
					Delta: 333,
				},
			},
			want: want{
				argResponse: &entities.Metric{
					Type:  entities.MetricTypeCounter,
					Name:  "Counter3",
					Delta: 333,
				},
				argError: nil,
			},
		},
		{
			name: "increase counter",
			given: given{
				argMetric: entities.Metric{
					Type:  entities.MetricTypeCounter,
					Name:  "Counter2",
					Delta: 2000,
				},
			},
			want: want{
				argResponse: &entities.Metric{
					Type:  entities.MetricTypeCounter,
					Name:  "Counter2",
					Delta: 2222,
				},
				argError: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := filledMemStorage().UpdateMetric(
				context.Background(), tt.given.argMetric)
			assert.Equal(t, tt.want.argResponse, response)
			assert.Equal(t, err, tt.want.argError)
		})
	}
}

func TestMemStorage_UpdateMetrics(t *testing.T) {
	type given struct {
		argMetrics []entities.Metric
	}
	type want struct {
		argResponse []entities.Metric
		argError    error
	}
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "add and replace gauge",
			given: given{
				argMetrics: []entities.Metric{
					{
						Type:  entities.MetricTypeGauge,
						Name:  "Gauge3",
						Value: 3.33,
					}, {
						Type:  entities.MetricTypeGauge,
						Name:  "Gauge2",
						Value: 22.22,
					},
				},
			},
			want: want{
				argResponse: []entities.Metric{
					{
						Type:  entities.MetricTypeGauge,
						Name:  "Gauge3",
						Value: 3.33,
					}, {
						Type:  entities.MetricTypeGauge,
						Name:  "Gauge2",
						Value: 22.22,
					},
				},
				argError: nil,
			},
		}, {
			name: "add and increase counter",
			given: given{
				argMetrics: []entities.Metric{
					{
						Type:  entities.MetricTypeCounter,
						Name:  "Counter3",
						Delta: 333,
					}, {
						Type:  entities.MetricTypeCounter,
						Name:  "Counter2",
						Delta: 2000,
					},
				},
			},
			want: want{
				argResponse: []entities.Metric{
					{
						Type:  entities.MetricTypeCounter,
						Name:  "Counter3",
						Delta: 333,
					}, {
						Type:  entities.MetricTypeCounter,
						Name:  "Counter2",
						Delta: 2222,
					},
				},
				argError: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			response, err := filledMemStorage().UpdateMetrics(
				context.Background(), tt.given.argMetrics)
			assert.Equal(t, tt.want.argResponse, response)
			assert.Equal(t, err, tt.want.argError)
		})
	}
}
