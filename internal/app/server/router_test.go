package server

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PiskarevSA/go-advanced/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockUpdateParams struct {
	// input
	metricType  string
	metricName  string
	metricValue string
	// output
	error error
}

type mockGetParams struct {
	// input
	metricType string
	metricName string
	// output
	value string
	error error
}

type mockUsecase struct {
	t      *testing.T
	method string
	update mockUpdateParams
	get    mockGetParams
	called bool
}

func expectNothing(t *testing.T) *mockUsecase {
	return &mockUsecase{
		t:      t,
		method: "Nothing",
		called: true,
	}
}

func expectUpdate(t *testing.T, metricType string, metricName string, metricValue string, error error) *mockUsecase {
	return &mockUsecase{
		t:      t,
		method: "Update",
		update: mockUpdateParams{
			metricType:  metricType,
			metricName:  metricName,
			metricValue: metricValue,
			error:       error,
		},
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

func (m *mockUsecase) Update(metricType string, metricName string, metricValue string) error {
	assert.Equal(m.t, m.method, "Update")
	assert.Equal(m.t, m.update.metricType, metricType)
	assert.Equal(m.t, m.update.metricName, metricName)
	assert.Equal(m.t, m.update.metricValue, metricValue)
	m.called = true
	return m.update.error
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
				mockUsecase: expectUpdate(t, "gauge", "foo", "1.23", nil),
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
				mockUsecase: expectUpdate(t, "counter", "bar", "456", nil),
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
				method: http.MethodPost,
				url:    "/update/foo/123/456",
				mockUsecase: expectUpdate(t, "foo", "123", "456",
					errors.NewInvalidMetricTypeError("foo")),
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
			name: "update: incorrect gauge value",
			given: given{
				method: http.MethodPost,
				url:    "/update/gauge/foo/str_value",
				mockUsecase: expectUpdate(t, "gauge", "foo", "str_value",
					errors.NewMetricValueIsNotValidError(fmt.Errorf("parsing error"))),
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
				mockUsecase: expectUpdate(t, "counter", "foo", "str_value",
					errors.NewMetricValueIsNotValidError(fmt.Errorf("parsing error"))),
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
