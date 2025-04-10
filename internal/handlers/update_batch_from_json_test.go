package handlers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateBatchFromJSON(t *testing.T) {
	type given struct {
		method      string
		url         string
		body        string
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
			name: "update batch: positive",
			given: given{
				method: http.MethodPost,
				url:    "/updates/",
				body:   `[{"id":"foo","type":"gauge","value":1.23},{"id":"bar","type":"counter","delta":456}]`,
				mockUsecase: &mockMetricsUsecase{
					UpdateMetricsFunc: func(ctx context.Context, metrics []entities.Metric,
					) ([]entities.Metric, error) {
						expectedMetrics := []entities.Metric{
							{
								Type:  entities.MetricTypeGauge,
								Name:  "foo",
								Value: 1.23,
							},
							{
								Type:  entities.MetricTypeCounter,
								Name:  "bar",
								Delta: 456,
							},
						}
						require.Equal(t, expectedMetrics, metrics)
						return []entities.Metric{
							{
								Type:  entities.MetricTypeGauge,
								Name:  "foo",
								Value: 1.23,
							}, {
								Type:  entities.MetricTypeCounter,
								Name:  "bar",
								Delta: 456,
							},
						}, nil
					},
				},
			},
			want: want{
				code:        http.StatusOK,
				response:    `[{"id":"foo","type":"gauge","value":1.23},{"id":"bar","type":"counter","delta":456}]`,
				contentType: "application/json",
				callCount:   1,
			},
		},
		{
			name: "update batch: invalid method",
			given: given{
				method:      http.MethodPatch,
				url:         "/updates/",
				body:        `[{"id":"foo","type":"gauge","delta":1.23}]`,
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
			name: "update batch: empty metric type",
			given: given{
				method:      http.MethodPost,
				url:         "/updates/",
				body:        `[{}]`,
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "metric[0]: invalid metric type:",
				contentType: "text/plain; charset=utf-8",
				callCount:   0,
			},
		},
		{
			name: "update batch: unexpected metric type",
			given: given{
				method:      http.MethodPost,
				url:         "/updates/",
				body:        `[{"type":"foo"}]`,
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "metric[0]: invalid metric type: foo",
				contentType: "text/plain; charset=utf-8",
				callCount:   0,
			},
		},
		{
			name: "update batch: empty metric name",
			given: given{
				method:      http.MethodPost,
				url:         "/updates/",
				body:        `[{"type":"gauge"}]`,
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found",
				contentType: "text/plain; charset=utf-8",
				callCount:   0,
			},
		},
		{
			name: "update batch: empty metric value",
			given: given{
				method:      http.MethodPost,
				url:         "/updates/",
				body:        `[{"id":"foo","type":"gauge"}]`,
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "metric[0]: missing value",
				contentType: "text/plain; charset=utf-8",
				callCount:   0,
			},
		},
		{
			name: "update batch: incorrect gauge value",
			given: given{
				method:      http.MethodPost,
				url:         "/updates/",
				body:        `[{"id":"foo","type":"gauge","value":"str_value"}]`,
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "json request decoding: json: cannot unmarshal string into Go struct field Metric.value of type float64",
				contentType: "text/plain; charset=utf-8",
				callCount:   0,
			},
		},
		{
			name: "update batch: incorrect counter value",
			given: given{
				method:      http.MethodPost,
				url:         "/updates/",
				body:        `[{"id":"foo","type":"counter","delta":"str_value"}]`,
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "json request decoding: json: cannot unmarshal string into Go struct field Metric.delta of type int64",
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

			respCode, respContentType, respBody := testRequestJSON(
				t, ts, tt.given.method, tt.given.url, tt.given.body)
			// проверяем параметры ответа
			assert.Equal(t, tt.want.code, respCode)
			assert.Equal(t, tt.want.contentType, respContentType)
			assert.Equal(t, tt.want.response, strings.TrimSpace(respBody))
			assert.Equal(t, tt.want.callCount, len(tt.given.mockUsecase.calls.UpdateMetrics))
		})
	}
}
