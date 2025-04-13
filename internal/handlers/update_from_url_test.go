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

func TestUpdateFromURL(t *testing.T) {
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
			name: "update: gauge positive",
			given: given{
				method: http.MethodPost,
				url:    "/update/gauge/foo/1.23",
				mockUsecase: &mockMetricsUsecase{
					UpdateMetricFunc: func(ctx context.Context, metric entities.Metric,
					) (*entities.Metric, error) {
						require.Equal(t, entities.MetricTypeGauge, metric.Type)
						require.Equal(t, entities.MetricName("foo"), metric.Name)
						require.Equal(t, entities.Gauge(1.23), metric.Value)
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
				response:    "",
				contentType: "text/plain; charset=utf-8",
				callCount:   1,
			},
		},
		{
			name: "update: counter positive",
			given: given{
				method: http.MethodPost,
				url:    "/update/counter/bar/456",
				mockUsecase: &mockMetricsUsecase{
					UpdateMetricFunc: func(ctx context.Context, metric entities.Metric,
					) (*entities.Metric, error) {
						require.Equal(t, entities.MetricTypeCounter, metric.Type)
						require.Equal(t, entities.MetricName("bar"), metric.Name)
						require.Equal(t, entities.Counter(456), metric.Delta)
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
				response:    "",
				contentType: "text/plain; charset=utf-8",
				callCount:   1,
			},
		},
		{
			name: "update: invalid method",
			given: given{
				method:      http.MethodPatch,
				url:         "/update/gauge/foo/1.23",
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
			name: "update: unexpected metric type",
			given: given{
				method:      http.MethodPost,
				url:         "/update/foo/123/456",
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric type: foo\n",
				contentType: "text/plain; charset=utf-8",
				callCount:   0,
			},
		},
		{
			name: "update: empty metric name",
			given: given{
				method:      http.MethodPost,
				url:         "/update/gauge",
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
			name: "update: empty metric value",
			given: given{
				method:      http.MethodPost,
				url:         "/update/gauge/foo",
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
			name: "update: incorrect gauge value",
			given: given{
				method:      http.MethodPost,
				url:         "/update/gauge/foo/str_value",
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value: strconv.ParseFloat: parsing \"str_value\": invalid syntax\n",
				contentType: "text/plain; charset=utf-8",
				callCount:   0,
			},
		},
		{
			name: "update: incorrect counter value",
			given: given{
				method:      http.MethodPost,
				url:         "/update/counter/foo/str_value",
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value: strconv.ParseInt: parsing \"str_value\": invalid syntax\n",
				contentType: "text/plain; charset=utf-8",
				callCount:   0,
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
			assert.Equal(t, tt.want.callCount, len(tt.given.mockUsecase.calls.UpdateMetric))
		})
	}
}
