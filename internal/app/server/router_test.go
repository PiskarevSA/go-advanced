package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PiskarevSA/go-advanced/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockGetParams struct {
	metricType string
	metricName string
	value      string
	error      error
}

type mockUsecase struct {
	t            *testing.T
	method       string
	gaugeKey     string
	gaugeValue   float64
	get          mockGetParams
	counterKey   string
	counterValue int64
	called       bool
}

func expectNothing(t *testing.T) *mockUsecase {
	return &mockUsecase{
		t:      t,
		method: "Nothing",
		called: true,
	}
}

func expectSetGauge(t *testing.T, key string, value float64) *mockUsecase {
	return &mockUsecase{
		t:          t,
		method:     "SetGauge",
		gaugeKey:   key,
		gaugeValue: value,
	}
}

func expectGet(t *testing.T, metricType string, metricName string,
	value string, error error,
) *mockUsecase {
	return &mockUsecase{
		t:      t,
		method: "Get",
		get: mockGetParams{
			metricType: metricType,
			metricName: metricName,
			error:      error,
			value:      value,
		},
	}
}

func expectSetCounter(t *testing.T, key string, value int64) *mockUsecase {
	return &mockUsecase{
		t:            t,
		method:       "SetCounter",
		counterKey:   key,
		counterValue: value,
	}
}

func (m *mockUsecase) SetGauge(key string, value float64) {
	assert.Equal(m.t, m.method, "SetGauge")
	assert.Equal(m.t, m.gaugeKey, key)
	assert.Equal(m.t, m.gaugeValue, value)
	m.called = true
}

func (m *mockUsecase) Get(metricType string, metricName string) (
	value string, err error,
) {
	assert.Equal(m.t, m.method, "Get")
	assert.Equal(m.t, m.get.metricType, metricType)
	assert.Equal(m.t, m.get.metricName, metricName)
	m.called = true
	return m.get.value, m.get.error
}

func (m *mockUsecase) SetCounter(key string, value int64) {
	assert.Equal(m.t, m.method, "SetCounter")
	assert.Equal(m.t, m.counterKey, key)
	assert.Equal(m.t, m.counterValue, value)
	m.called = true
}

func (m *mockUsecase) Dump() (gauge map[string]float64, counter map[string]int64) {
	require.Fail(m.t, "unexpected call")
	m.called = true
	return
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
				method:      http.MethodPost,
				url:         "/update/gauge/foo/1.23",
				mockUsecase: expectSetGauge(t, "foo", 1.23),
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
				method:      http.MethodPost,
				url:         "/update/counter/bar/456",
				mockUsecase: expectSetCounter(t, "bar", 456),
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
				mockUsecase: expectNothing(t),
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
				mockUsecase: expectNothing(t),
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
				method:      http.MethodPost,
				url:         "/update/foo/123/456",
				mockUsecase: expectNothing(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "unexpected metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: empty metric name",
			given: given{
				method:      http.MethodPost,
				url:         "/update/gauge",
				mockUsecase: expectNothing(t),
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
				mockUsecase: expectNothing(t),
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: incorrect metric value",
			given: given{
				method:      http.MethodPost,
				url:         "/update/gauge/foo/qwe",
				mockUsecase: expectNothing(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "incorrect metric value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "update: incorrect metric value",
			given: given{
				method:      http.MethodPost,
				url:         "/update/gauge/foo/value",
				mockUsecase: expectNothing(t),
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "incorrect metric value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "value: gauge positive",
			given: given{
				method:      http.MethodGet,
				url:         "/value/gauge/foo",
				mockUsecase: expectGet(t, "gauge", "foo", "1.23", nil),
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
				method:      http.MethodGet,
				url:         "/value/counter/bar",
				mockUsecase: expectGet(t, "counter", "bar", "456", nil),
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
				mockUsecase: expectNothing(t),
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
				mockUsecase: expectNothing(t),
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
				mockUsecase: expectGet(t, "gauge", "foo",
					"", errors.NewMetricNameNotFoundError("foo")),
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
			if tt.given.mockUsecase.method != "Nothing" {
				assert.Truef(t, tt.given.mockUsecase.called, "call of %v expected", tt.given.mockUsecase.method)
			}
		})
	}
}
