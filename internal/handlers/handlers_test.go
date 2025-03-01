package handlers

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockRepo struct {
	t               *testing.T
	gaugeKey        string
	gaugeValue      float64
	counterKey      string
	counterAddition int64
}

func expectSetGauge(t *testing.T, key string, value float64) *mockRepo {
	return &mockRepo{
		t:          t,
		gaugeKey:   key,
		gaugeValue: value,
	}
}

func expectIncreaseCounter(t *testing.T, key string, addition int64) *mockRepo {
	return &mockRepo{
		t:               t,
		counterKey:      key,
		counterAddition: addition,
	}
}

func (m *mockRepo) SetGauge(key string, value float64) {
	assert.Equal(m.t, m.gaugeKey, key)
	assert.Equal(m.t, m.gaugeValue, value)
}

func (m *mockRepo) IncreaseCounter(key string, addition int64) {
	assert.Equal(m.t, m.counterKey, key)
	assert.Equal(m.t, m.counterAddition, addition)
}

func TestUpdate(t *testing.T) {
	type given struct {
		method      string
		contentType string
		url         string
		mockRepo    *mockRepo
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
			name: "gauge positive",
			given: given{
				method:      http.MethodPost,
				contentType: "text/plain",
				url:         "http://localhost:8080/update/gauge/foo/1.23",
				mockRepo:    expectSetGauge(t, "foo", 1.23),
			},
			want: want{
				code:        http.StatusOK,
				response:    "",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "counter positive",
			given: given{
				method:      http.MethodPost,
				contentType: "text/plain",
				url:         "http://localhost:8080/update/counter/bar/456",
				mockRepo:    expectIncreaseCounter(t, "bar", 456),
			},
			want: want{
				code:        http.StatusOK,
				response:    "",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "invalid method",
			given: given{
				method:      http.MethodPatch,
				contentType: "text/plain",
				url:         "http://localhost:8080/update/gauge/foo/1.23",
				mockRepo:    nil,
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "invalid content type",
			given: given{
				method:      http.MethodPost,
				contentType: "application/json",
				url:         "http://localhost:8080/update/gauge/foo/1.23",
				mockRepo:    nil,
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "supported Content-Type: text/plain\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "empty metric type",
			given: given{
				method:      http.MethodPost,
				contentType: "text/plain",
				url:         "http://localhost:8080/update/",
				mockRepo:    nil,
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "empty metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "unexpected metric type",
			given: given{
				method:      http.MethodPost,
				contentType: "text/plain",
				url:         "http://localhost:8080/update/foo",
				mockRepo:    nil,
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "unexpected metric type\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "empty metric name",
			given: given{
				method:      http.MethodPost,
				contentType: "text/plain",
				url:         "http://localhost:8080/update/gauge",
				mockRepo:    nil,
			},
			want: want{
				code:        http.StatusNotFound,
				response:    "404 page not found\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "incorrect metric value",
			given: given{
				method:      http.MethodPost,
				contentType: "text/plain",
				url:         "http://localhost:8080/update/gauge/foo",
				mockRepo:    nil,
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "incorrect metric value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
		{
			name: "incorrect metric value",
			given: given{
				method:      http.MethodPost,
				contentType: "text/plain",
				url:         "http://localhost:8080/update/gauge/foo/value",
				mockRepo:    nil,
			},
			want: want{
				code:        http.StatusBadRequest,
				response:    "incorrect metric value\n",
				contentType: "text/plain; charset=utf-8",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.given.method, tt.given.url, http.NoBody)
			request.Header.Set("Content-Type", tt.given.contentType)
			// создаём новый Recorder
			w := httptest.NewRecorder()
			Update(tt.given.mockRepo)(w, request)

			res := w.Result()
			// проверяем код ответа
			assert.Equal(t, tt.want.code, res.StatusCode)
			// получаем и проверяем тело запроса
			defer res.Body.Close()
			resBody, err := io.ReadAll(res.Body)

			require.NoError(t, err)
			assert.Equal(t, tt.want.response, string(resBody))
			assert.Equal(t, tt.want.contentType, res.Header.Get("Content-Type"))
		})
	}
}
