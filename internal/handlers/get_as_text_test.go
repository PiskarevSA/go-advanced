package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAsText(t *testing.T) {
	type given struct {
		method      string
		url         string
		mockUsecase *mockMetricsUsecase
	}
	type want struct {
		code        int
		response    string
		contentType string
		callCount   int
	}
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "value: gauge positive",
			given: given{
				method: http.MethodGet,
				url:    "/value/gauge/foo",
				mockUsecase: &mockMetricsUsecase{
					GetMetricFunc: func(ctx context.Context, metric entities.Metric,
					) (*entities.Metric, error) {
						require.Equal(t, entities.MetricTypeGauge, metric.Type)
						require.Equal(t, entities.MetricName("foo"), metric.Name)
						return &entities.Metric{
							Type:  entities.MetricTypeGauge,
							Name:  "foo",
							Value: 1.23,
						}, nil
					},
				},
			},
			want: want{
				code:        http.StatusOK,
				response:    "1.23",
				contentType: "text/plain; charset=utf-8",
				callCount:   1,
			},
		},
		{
			name: "value: counter positive",
			given: given{
				method: http.MethodGet,
				url:    "/value/counter/bar",
				mockUsecase: &mockMetricsUsecase{
					GetMetricFunc: func(ctx context.Context, metric entities.Metric,
					) (*entities.Metric, error) {
						require.Equal(t, entities.MetricTypeCounter, metric.Type)
						require.Equal(t, entities.MetricName("bar"), metric.Name)
						return &entities.Metric{
							Type:  entities.MetricTypeCounter,
							Name:  "bar",
							Delta: 456,
						}, nil
					},
				},
			},
			want: want{
				code:        http.StatusOK,
				response:    "456",
				contentType: "text/plain; charset=utf-8",
				callCount:   1,
			},
		},
		{
			name: "value: invalid method",
			given: given{
				method:      http.MethodPatch,
				url:         "/value/gauge/foo",
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
				callCount:   0,
			},
		},
		{
			name: "value: unknown metric type",
			given: given{
				method:      http.MethodGet,
				url:         "/value/foo",
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
				callCount:   0,
			},
		},
		{
			name: "value: unknown metric name",
			given: given{
				method: http.MethodGet,
				url:    "/value/gauge/foo",
				mockUsecase: &mockMetricsUsecase{
					GetMetricFunc: func(ctx context.Context, metric entities.Metric,
					) (*entities.Metric, error) {
						require.Equal(t, entities.MetricTypeGauge, metric.Type)
						require.Equal(t, entities.MetricName("foo"), metric.Name)
						return nil, entities.NewMetricNameNotFoundError("foo")
					},
				},
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
				callCount:   1,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := NewMetricsRouter(tt.given.mockUsecase).WithAllHandlers()
			ts := httptest.NewServer(r)
			defer ts.Close()

			respCode, respContentType, respBody := testRequest(
				t, ts, tt.given.method, tt.given.url)
			// проверяем параметры ответа
			assert.Equal(t, tt.want.code, respCode)
			assert.Equal(t, tt.want.contentType, respContentType)
			assert.Equal(t, tt.want.response, respBody)
			assert.Equal(t, tt.want.callCount, len(tt.given.mockUsecase.calls.GetMetric))
		})
	}
}
