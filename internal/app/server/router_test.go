package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockGetArgs struct {
	// input
	metric entities.Metric
	// output
	result *entities.Metric
	err    error
}

type mockUpdateArgs struct {
	// input
	metric entities.Metric
	// output
	result *entities.Metric
	err    error
}

type mockUsecase struct {
	t              *testing.T
	mockCallParams []any
	callIndex      int
}

func newMockUsecase(t *testing.T) *mockUsecase {
	return &mockUsecase{
		t: t,
	}
}

func (m *mockUsecase) expectCall(mockCallParams any) *mockUsecase {
	m.mockCallParams = append(m.mockCallParams, mockCallParams)
	return m
}

func (m *mockUsecase) Get(metric entities.Metric) (*entities.Metric, error) {
	require.Less(m.t, m.callIndex, len(m.mockCallParams))
	args := m.mockCallParams[m.callIndex].(mockGetArgs)
	assert.Equal(m.t, args.metric, metric)
	m.callIndex += 1
	return args.result, args.err
}

func (m *mockUsecase) Update(metric entities.Metric) (*entities.Metric, error) {
	require.Less(m.t, m.callIndex, len(m.mockCallParams))
	args := m.mockCallParams[m.callIndex].(mockUpdateArgs)
	assert.Equal(m.t, args.metric, metric)
	m.callIndex += 1
	return args.result, args.err
}

func (m *mockUsecase) DumpIterator() func() (type_ string, name string, value string, exists bool) {
	require.Fail(m.t, "unexpected call")
	return nil
}

func testRequestJSON(t *testing.T, ts *httptest.Server, method, path string, body string) (
	respCode int, respContentType, respBody string,
) {
	bodyReader := strings.NewReader(body)
	req, err := http.NewRequest(method, ts.URL+path, bodyReader)
	req.Header.Set("Content-Type", "application/json")
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, resp.Header.Get("Content-Type"), string(respBodyBytes)
}

func testRequest(t *testing.T, ts *httptest.Server, method, path string) (
	respCode int, respContentType, respBody string,
) {
	req, err := http.NewRequest(method, ts.URL+path, http.NoBody)
	require.NoError(t, err)

	resp, err := ts.Client().Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	respBodyBytes, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, resp.Header.Get("Content-Type"), string(respBodyBytes)
}

func TestMetricsRouterJSON(t *testing.T) {
	type given struct {
		method      string
		url         string
		body        string
		mockUsecase *mockUsecase
	}
	type want struct {
		code        int
		response    string
		contentType string
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
				url:    "/update/",
				body:   `{"id":"foo","type":"gauge","value":1.23}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockUpdateArgs{
						// input
						metric: entities.Metric{
							Type:  entities.MetricTypeGauge,
							Name:  "foo",
							Value: 1.23,
						},
						// output
						result: &entities.Metric{
							Type:  entities.MetricTypeGauge,
							Name:  "foo",
							Value: 1.23,
						},
						err: nil,
					}),
			},
			want: want{
				code:        http.StatusOK,
				response:    `{"id":"foo","type":"gauge","value":1.23}`,
				contentType: "application/json",
			},
		},
		{
			name: "update: counter positive",
			given: given{
				method: http.MethodPost,
				url:    "/update/",
				body:   `{"id":"bar","type":"counter","delta":456}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockUpdateArgs{
						// input
						metric: entities.Metric{
							Type:  entities.MetricTypeCounter,
							Name:  "bar",
							Delta: 456,
						},
						// output
						result: &entities.Metric{
							Type:  entities.MetricTypeCounter,
							Name:  "bar",
							Delta: 456,
						},
						err: nil,
					}),
			},
			want: want{
				code:        http.StatusOK,
				response:    `{"id":"bar","type":"counter","delta":456}`,
				contentType: "application/json",
			},
		},
		{
			name: "update: invalid method",
			given: given{
				method:      http.MethodPatch,
				url:         "/update/",
				body:        `{"id":"foo","type":"gauge","delta":1.23}`,
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
			},
		},
		{
			name: "update: empty metric type",
			given: given{
				method:      http.MethodPost,
				url:         "/update/",
				body:        `{}`,
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric type:",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: unexpected metric type",
			given: given{
				method:      http.MethodPost,
				url:         "/update/",
				body:        `{"type":"foo"}`,
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric type: foo",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: empty metric name",
			given: given{
				method:      http.MethodPost,
				url:         "/update/",
				body:        `{"type":"gauge"}`,
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: empty metric value",
			given: given{
				method:      http.MethodPost,
				url:         "/update/",
				body:        `{"id":"foo","type":"gauge"}`,
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "missing value",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: incorrect gauge value",
			given: given{
				method:      http.MethodPost,
				url:         "/update/",
				body:        `{"id":"foo","type":"gauge","value":"str_value"}`,
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "json request decoding: json: cannot unmarshal string into Go struct field Metric.value of type float64",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: incorrect counter value",
			given: given{
				method:      http.MethodPost,
				url:         "/update/",
				body:        `{"id":"foo","type":"counter","delta":"str_value"}`,
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "json request decoding: json: cannot unmarshal string into Go struct field Metric.delta of type int64",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "value: gauge positive",
			given: given{
				method: http.MethodPost,
				url:    "/value/",
				body:   `{"id":"foo","type":"gauge"}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockGetArgs{
						// input
						metric: entities.Metric{
							Type: entities.MetricTypeGauge,
							Name: "foo",
						},
						// output
						result: &entities.Metric{
							Type:  entities.MetricTypeGauge,
							Name:  "foo",
							Value: 1.23,
						},
						err: nil,
					}),
			},
			want: want{
				code:        http.StatusOK,
				response:    `{"id":"foo","type":"gauge","value":1.23}`,
				contentType: "application/json",
			},
		},
		{
			name: "value: counter positive",
			given: given{
				method: http.MethodPost,
				url:    "/value/",
				body:   `{"id":"bar","type":"counter"}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockGetArgs{
						// input
						metric: entities.Metric{
							Type: entities.MetricTypeCounter,
							Name: "bar",
						},
						// output
						result: &entities.Metric{
							Type:  entities.MetricTypeCounter,
							Name:  "bar",
							Delta: 456,
						},
						err: nil,
					}),
			},
			want: want{
				code:        http.StatusOK,
				response:    `{"id":"bar","type":"counter","delta":456}`,
				contentType: "application/json",
			},
		},
		{
			name: "value: invalid method",
			given: given{
				method:      http.MethodPatch,
				url:         "/value/",
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
			},
		},
		{
			name: "value: unknown metric type",
			given: given{
				method:      http.MethodPost,
				url:         "/value/",
				body:        `{"type":"foo"}`,
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "value: unknown metric name",
			given: given{
				method: http.MethodPost,
				url:    "/value/",
				body:   `{"id":"foo","type":"gauge"}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockGetArgs{
						// input
						metric: entities.Metric{
							Type: entities.MetricTypeGauge,
							Name: "foo",
						},
						// output
						result: &entities.Metric{
							Type:  entities.MetricTypeGauge,
							Name:  "foo",
							Value: 0,
						},
						err: entities.NewMetricNameNotFoundError("foo"),
					}),
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(MetricsRouter(tt.given.mockUsecase))
			defer ts.Close()

			respCode, respContentType, respBody := testRequestJSON(
				t, ts, tt.given.method, tt.given.url, tt.given.body)
			// проверяем параметры ответа
			assert.Equal(t, tt.want.code, respCode)
			assert.Equal(t, tt.want.contentType, respContentType)
			assert.Equal(t, tt.want.response, strings.TrimSpace(respBody))
			assert.Equal(t, len(tt.given.mockUsecase.mockCallParams),
				tt.given.mockUsecase.callIndex)
		})
	}
}

func TestMetricsRouter(t *testing.T) {
	type given struct {
		method      string
		url         string
		mockUsecase *mockUsecase
	}
	type want struct {
		code        int
		response    string
		contentType string
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
				mockUsecase: newMockUsecase(t).
					expectCall(mockUpdateArgs{
						// input
						metric: entities.Metric{
							Type:  entities.MetricTypeGauge,
							Name:  "foo",
							Value: 1.23,
						},
						// output
						result: &entities.Metric{
							Type:  entities.MetricTypeGauge,
							Name:  "foo",
							Value: 1.23,
						},
						err: nil,
					}),
			},
			want: want{
				code:        http.StatusOK,
				response:    "",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: counter positive",
			given: given{
				method: http.MethodPost,
				url:    "/update/counter/bar/456",
				mockUsecase: newMockUsecase(t).
					expectCall(mockUpdateArgs{
						// input
						metric: entities.Metric{
							Type:  entities.MetricTypeCounter,
							Name:  "bar",
							Delta: 456,
						},
						// output
						result: &entities.Metric{
							Type:  entities.MetricTypeCounter,
							Name:  "bar",
							Delta: 456,
						},
						err: nil,
					}),
			},
			want: want{
				code:        http.StatusOK,
				response:    "",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: invalid method",
			given: given{
				method:      http.MethodPatch,
				url:         "/update/gauge/foo/1.23",
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
			},
		},
		{
			name: "update: unexpected metric type",
			given: given{
				method:      http.MethodPost,
				url:         "/update/foo/123/456",
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric type: foo\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: empty metric name",
			given: given{
				method:      http.MethodPost,
				url:         "/update/gauge",
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: empty metric value",
			given: given{
				method:      http.MethodPost,
				url:         "/update/gauge/foo",
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: incorrect gauge value",
			given: given{
				method:      http.MethodPost,
				url:         "/update/gauge/foo/str_value",
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value: strconv.ParseFloat: parsing \"str_value\": invalid syntax\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: incorrect counter value",
			given: given{
				method:      http.MethodPost,
				url:         "/update/counter/foo/str_value",
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value: strconv.ParseInt: parsing \"str_value\": invalid syntax\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "value: gauge positive",
			given: given{
				method: http.MethodGet,
				url:    "/value/gauge/foo",
				mockUsecase: newMockUsecase(t).
					expectCall(mockGetArgs{
						// input
						metric: entities.Metric{
							Type: entities.MetricTypeGauge,
							Name: "foo",
						},
						// output
						result: &entities.Metric{
							Type:  entities.MetricTypeGauge,
							Name:  "foo",
							Value: 1.23,
						},
						err: nil,
					}),
			},
			want: want{
				code:        http.StatusOK,
				response:    "1.23",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "value: counter positive",
			given: given{
				method: http.MethodGet,
				url:    "/value/counter/bar",
				mockUsecase: newMockUsecase(t).
					expectCall(mockGetArgs{
						// input
						metric: entities.Metric{
							Type: entities.MetricTypeCounter,
							Name: "bar",
						},
						// output
						result: &entities.Metric{
							Type:  entities.MetricTypeCounter,
							Name:  "bar",
							Delta: 456,
						},
						err: nil,
					}),
			},
			want: want{
				code:        http.StatusOK,
				response:    "456",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "value: invalid method",
			given: given{
				method:      http.MethodPatch,
				url:         "/value/gauge/foo",
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
			},
		},
		{
			name: "value: unknown metric type",
			given: given{
				method:      http.MethodGet,
				url:         "/value/foo",
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "value: unknown metric name",
			given: given{
				method: http.MethodGet,
				url:    "/value/gauge/foo",
				mockUsecase: newMockUsecase(t).
					expectCall(mockGetArgs{
						// input
						metric: entities.Metric{
							Type: entities.MetricTypeGauge,
							Name: "foo",
						},
						// output
						result: nil,
						err:    entities.NewMetricNameNotFoundError("foo"),
					}),
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := httptest.NewServer(MetricsRouter(tt.given.mockUsecase))
			defer ts.Close()

			respCode, respContentType, respBody := testRequest(
				t, ts, tt.given.method, tt.given.url)
			// проверяем параметры ответа
			assert.Equal(t, tt.want.code, respCode)
			assert.Equal(t, tt.want.contentType, respContentType)
			assert.Equal(t, tt.want.response, respBody)
			assert.Equal(t, len(tt.given.mockUsecase.mockCallParams),
				tt.given.mockUsecase.callIndex)
		})
	}
}
