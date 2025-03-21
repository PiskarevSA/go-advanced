package server

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/PiskarevSA/go-advanced/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockIsGaugeArgs struct {
	// input
	metricType string
	// output
	result bool
	err    error
}

type mockUpdateArgs struct {
	// input
	metricType  string
	metricName  string
	metricValue string
	// output
	err error
}

type mockUpdateGaugeArgs struct {
	// input
	metricName string
	value      *float64
	// output
	err error
}

type mockIncreaseCounterArgs struct {
	// input
	metricName string
	delta      *int64
	// output
	result *int64
	err    error
}

type mockGetArgs struct {
	// input
	metricType string
	metricName string
	// output
	result string
	err    error
}

type mockGetGaugeArgs struct {
	// input
	metricName string
	// output
	result *float64
	err    error
}

type mockGetCounterArgs struct {
	// input
	metricName string
	// output
	result *int64
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

func (m *mockUsecase) IsGauge(metricType string) (bool, error) {
	require.Less(m.t, m.callIndex, len(m.mockCallParams))
	args := m.mockCallParams[m.callIndex].(mockIsGaugeArgs)
	assert.Equal(m.t, args.metricType, metricType)
	m.callIndex += 1
	return args.result, args.err
}

func (m *mockUsecase) Update(metricType string, metricName string, metricValue string) error {
	require.Less(m.t, m.callIndex, len(m.mockCallParams))
	args := m.mockCallParams[m.callIndex].(mockUpdateArgs)
	assert.Equal(m.t, args.metricType, metricType)
	assert.Equal(m.t, args.metricName, metricName)
	assert.Equal(m.t, args.metricValue, metricValue)
	m.callIndex += 1
	return args.err
}

func (m *mockUsecase) UpdateGauge(metricName string, value *float64) error {
	require.Less(m.t, m.callIndex, len(m.mockCallParams))
	args := m.mockCallParams[m.callIndex].(mockUpdateGaugeArgs)
	assert.Equal(m.t, args.metricName, metricName)
	assert.Equal(m.t, args.value, value)
	m.callIndex += 1
	return args.err
}

func (m *mockUsecase) IncreaseCounter(metricName string, delta *int64) (value *int64, err error) {
	require.Less(m.t, m.callIndex, len(m.mockCallParams))
	args := m.mockCallParams[m.callIndex].(mockIncreaseCounterArgs)
	assert.Equal(m.t, args.metricName, metricName)
	assert.Equal(m.t, args.delta, delta)
	m.callIndex += 1
	return args.result, args.err
}

func (m *mockUsecase) Get(metricType string, metricName string) (value string, err error) {
	require.Less(m.t, m.callIndex, len(m.mockCallParams))
	args := m.mockCallParams[m.callIndex].(mockGetArgs)
	assert.Equal(m.t, args.metricType, metricType)
	assert.Equal(m.t, args.metricName, metricName)
	m.callIndex += 1
	return args.result, args.err
}

func (m *mockUsecase) GetGauge(metricName string) (value *float64, err error) {
	require.Less(m.t, m.callIndex, len(m.mockCallParams))
	args := m.mockCallParams[m.callIndex].(mockGetGaugeArgs)
	assert.Equal(m.t, args.metricName, metricName)
	m.callIndex += 1
	return args.result, args.err
}

func (m *mockUsecase) GetCounter(metricName string) (value *int64, err error) {
	require.Less(m.t, m.callIndex, len(m.mockCallParams))
	args := m.mockCallParams[m.callIndex].(mockGetCounterArgs)
	assert.Equal(m.t, args.metricName, metricName)
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
	f64_1_23 := 1.23
	var i64_456 int64 = 456
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "update: gauge positive",
			given: given{
				method: http.MethodPost,
				url:    "/update",
				body:   `{"id":"foo","type":"gauge","value":1.23}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockIsGaugeArgs{
						// input
						metricType: "gauge",
						// output
						result: true,
						err:    nil,
					}).
					expectCall(mockUpdateGaugeArgs{
						// input
						metricName: "foo",
						value:      &f64_1_23,
						// output
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
				url:    "/update",
				body:   `{"id":"bar","type":"counter","delta":456}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockIsGaugeArgs{
						// input
						metricType: "counter",
						// output
						result: false,
						err:    nil,
					}).
					expectCall(mockIncreaseCounterArgs{
						// input
						metricName: "bar",
						delta:      &i64_456,
						// output
						result: &i64_456,
						err:    nil,
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
				url:         "/update",
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
				method: http.MethodPost,
				url:    "/update",
				body:   `{}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockIsGaugeArgs{
						// input
						metricType: "",
						// output
						result: false,
						err:    errors.NewEmptyMetricTypeError(),
					}),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "empty metric type",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: unexpected metric type",
			given: given{
				method: http.MethodPost,
				url:    "/update",
				body:   `{"type":"foo"}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockIsGaugeArgs{
						// input
						metricType: "foo",
						// output
						result: false,
						err:    errors.NewInvalidMetricTypeError("foo"),
					}),
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
				method: http.MethodPost,
				url:    "/update",
				body:   `{"type":"gauge"}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockIsGaugeArgs{
						// input
						metricType: "gauge",
						// output
						result: true,
						err:    nil,
					}).
					expectCall(mockUpdateGaugeArgs{
						// input
						metricName: "",
						value:      nil,
						// output
						err: errors.NewEmptyMetricNameError(),
					}),
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
				method: http.MethodPost,
				url:    "/update",
				body:   `{"id":"foo","type":"gauge"}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockIsGaugeArgs{
						// input
						metricType: "gauge",
						// output
						result: true,
						err:    nil,
					}).
					expectCall(mockUpdateGaugeArgs{
						// input
						metricName: "foo",
						value:      nil,
						// output
						err: errors.NewMissingValueError(),
					}),
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
				url:         "/update",
				body:        `{"id":"foo","type":"gauge","value":"str_value"}`,
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "json: cannot unmarshal string into Go struct field Metrics.value of type float64",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: incorrect counter value",
			given: given{
				method:      http.MethodPost,
				url:         "/update",
				body:        `{"id":"foo","type":"counter","delta":"str_value"}`,
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "json: cannot unmarshal string into Go struct field Metrics.delta of type int64",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "value: gauge positive",
			given: given{
				method: http.MethodPost,
				url:    "/value",
				body:   `{"id":"foo","type":"gauge"}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockIsGaugeArgs{
						// input
						metricType: "gauge",
						// output
						result: true,
						err:    nil,
					}).expectCall(mockGetGaugeArgs{
					// input
					metricName: "foo",
					// output
					result: &f64_1_23,
					err:    nil,
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
				url:    "/value",
				body:   `{"id":"bar","type":"counter"}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockIsGaugeArgs{
						// input
						metricType: "counter",
						// output
						result: false,
						err:    nil,
					}).expectCall(mockGetCounterArgs{
					// input
					metricName: "bar",
					// output
					result: &i64_456,
					err:    nil,
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
				url:         "/value",
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
				method: http.MethodPost,
				url:    "/value",
				body:   `{"type":"foo"}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockIsGaugeArgs{
						// input
						metricType: "foo",
						// output
						result: false,
						err:    errors.NewInvalidMetricTypeError("foo"),
					}),
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
				url:    "/value",
				body:   `{"id":"foo","type":"gauge"}`,
				mockUsecase: newMockUsecase(t).
					expectCall(mockIsGaugeArgs{
						// input
						metricType: "gauge",
						// output
						result: true,
						err:    nil,
					}).
					expectCall(mockGetGaugeArgs{
						// input
						metricName: "foo",
						// output
						result: nil,
						err:    errors.NewMetricNameNotFoundError("foo"),
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
						metricType:  "gauge",
						metricName:  "foo",
						metricValue: "1.23",
						// output
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
						metricType:  "counter",
						metricName:  "bar",
						metricValue: "456",
						// output
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
			name: "update: empty metric type",
			given: given{
				method:      http.MethodPost,
				url:         "/update/",
				mockUsecase: newMockUsecase(t),
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: unexpected metric type",
			given: given{
				method: http.MethodPost,
				url:    "/update/foo/123/456",
				mockUsecase: newMockUsecase(t).
					expectCall(mockUpdateArgs{
						// input
						metricType:  "foo",
						metricName:  "123",
						metricValue: "456",
						// output
						err: errors.NewInvalidMetricTypeError("foo"),
					}),
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
				method: http.MethodPost,
				url:    "/update/gauge/foo/str_value",
				mockUsecase: newMockUsecase(t).
					expectCall(mockUpdateArgs{
						// input
						metricType:  "gauge",
						metricName:  "foo",
						metricValue: "str_value",
						// output
						err: errors.NewMetricValueIsNotValidError(fmt.Errorf("parsing error")),
					}),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value: parsing error\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: incorrect counter value",
			given: given{
				method: http.MethodPost,
				url:    "/update/counter/foo/str_value",
				mockUsecase: newMockUsecase(t).
					expectCall(mockUpdateArgs{
						// input
						metricType:  "counter",
						metricName:  "foo",
						metricValue: "str_value",
						// output
						err: errors.NewMetricValueIsNotValidError(fmt.Errorf("parsing error")),
					}),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "invalid metric value: parsing error\n",
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
						metricType: "gauge",
						metricName: "foo",
						// output
						result: "1.23",
						err:    nil,
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
						metricType: "counter",
						metricName: "bar",
						// output
						result: "456",
						err:    nil,
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
						metricType: "gauge",
						metricName: "foo",
						// output
						result: "",
						err:    errors.NewMetricNameNotFoundError("foo"),
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
