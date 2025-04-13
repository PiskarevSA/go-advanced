package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPing(t *testing.T) {
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
			name: "ping: positive",
			given: given{
				method: http.MethodGet,
				url:    "/ping",
				mockUsecase: &mockMetricsUsecase{
					PingFunc: func(ctx context.Context) error { return nil },
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
			name: "ping: error",
			given: given{
				method: http.MethodGet,
				url:    "/ping",
				mockUsecase: &mockMetricsUsecase{
					PingFunc: func(ctx context.Context) error {
						return errors.New("some error")
					},
				},
			},
			want: want{
				code:        http.StatusInternalServerError,
				response:    "some error\n",
				contentType: "text/plain; charset=utf-8",
				callCount:   1,
			},
		},
		{
			name: "ping: method not allowed",
			given: given{
				method:      http.MethodPost,
				url:         "/ping",
				mockUsecase: &mockMetricsUsecase{},
			},
			want: want{
				code:        http.StatusMethodNotAllowed,
				response:    "",
				contentType: "",
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
			assert.Equal(t, tt.want.callCount, len(tt.given.mockUsecase.calls.Ping))
		})
	}
}
