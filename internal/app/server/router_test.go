package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUsecase struct {
	t             *testing.T
	method        string
	gaugeKey      string
	gaugeValue    float64
	gaugeExists   bool
	counterKey    string
	counterValue  int64
	counterExists bool
}

func expectSetGauge(t *testing.T, key string, value float64) *mockUsecase {
	return &mockUsecase{
		t:          t,
		method:     "SetGauge",
		gaugeKey:   key,
		gaugeValue: value,
	}
}

func expectGauge(t *testing.T, key string, value float64, exists bool) *mockUsecase {
	return &mockUsecase{
		t:           t,
		method:      "Gauge",
		gaugeKey:    key,
		gaugeValue:  value,
		gaugeExists: exists,
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

func expectCounter(t *testing.T, key string, value int64, exists bool) *mockUsecase {
	return &mockUsecase{
		t:             t,
		method:        "Counter",
		counterKey:    key,
		counterValue:  value,
		counterExists: exists,
	}
}

func (m *mockUsecase) SetGauge(key string, value float64) {
	assert.Equal(m.t, m.method, "SetGauge")
	assert.Equal(m.t, m.gaugeKey, key)
	assert.Equal(m.t, m.gaugeValue, value)
}

func (m *mockUsecase) Gauge(key string) (value float64, exist bool) {
	assert.Equal(m.t, m.method, "Gauge")
	assert.Equal(m.t, m.gaugeKey, key)
	return m.gaugeValue, m.gaugeExists
}

func (m *mockUsecase) SetCounter(key string, value int64) {
	assert.Equal(m.t, m.method, "SetCounter")
	assert.Equal(m.t, m.counterKey, key)
	assert.Equal(m.t, m.counterValue, value)
}

func (m *mockUsecase) Counter(key string) (value int64, exist bool) {
	assert.Equal(m.t, m.method, "Counter")
	assert.Equal(m.t, m.counterKey, key)
	return m.counterValue, m.counterExists
}

func (m *mockUsecase) Dump() (gauge map[string]float64, counter map[string]int64) {
	require.Fail(m.t, "unexpected call")
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
				mockUsecase: nil,
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
				mockUsecase: nil,
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
				mockUsecase: nil,
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
				mockUsecase: nil,
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
				mockUsecase: nil,
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
				mockUsecase: nil,
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
				mockUsecase: nil,
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
				mockUsecase: expectGauge(t, "foo", 1.23, true),
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
				mockUsecase: expectCounter(t, "bar", 456, true),
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
				mockUsecase: nil,
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
				mockUsecase: nil,
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
				method:      http.MethodGet,
				url:         "/value/gauge/foo",
				mockUsecase: expectGauge(t, "foo", 0.0, false),
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
		})
	}
}
