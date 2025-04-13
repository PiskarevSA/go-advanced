package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMainPage(t *testing.T) {
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
			name: "mainpage: empty",
			given: given{
				method: http.MethodGet,
				url:    "/",
				mockUsecase: &mockMetricsUsecase{
					DumpIteratorFunc: func(ctx context.Context) (
						func() (type_ string, name string, value string, exists bool), error,
					) {
						fn := func() (type_ string, name string, value string, exists bool) {
							return "", "", "", false
						}
						var err error
						return fn, err
					},
				},
			},
			want: want{
				code: http.StatusOK,
				response: `<!DOCTYPE html>
<title>Metrics</title>
<body>
	<table>
		<tr>
			<th>type</th>
			<th>key</th>
			<th>value</th>
		</tr>
	</table>
</body>
`,
				contentType: "text/html",
				callCount:   1,
			},
		},
		{
			name: "mainpage: filled",
			given: given{
				method: http.MethodGet,
				url:    "/",
				mockUsecase: &mockMetricsUsecase{
					DumpIteratorFunc: func(ctx context.Context) (
						func() (type_ string, name string, value string, exists bool), error,
					) {
						callNumber := 0
						fn := func() (type_ string, name string, value string, exists bool) {
							callNumber++
							switch callNumber {
							case 1:
								return "type1", "name1", "value1", true
							case 2:
								return "type2", "name2", "value2", true
							case 3:
								return "type3", "name3", "value3", true
							default:
								return "", "", "", false
							}
						}
						var err error
						return fn, err
					},
				},
			},
			want: want{
				code: http.StatusOK,
				response: `<!DOCTYPE html>
<title>Metrics</title>
<body>
	<table>
		<tr>
			<th>type</th>
			<th>key</th>
			<th>value</th>
		</tr>
		<tr>
			<td>type1</td>
			<td>name1</td>
			<td>value1</td>
		</tr>
		<tr>
			<td>type2</td>
			<td>name2</td>
			<td>value2</td>
		</tr>
		<tr>
			<td>type3</td>
			<td>name3</td>
			<td>value3</td>
		</tr>
	</table>
</body>
`,
				contentType: "text/html",
				callCount:   1,
			},
		},
		{
			name: "mainpage: some error",
			given: given{
				method: http.MethodGet,
				url:    "/",
				mockUsecase: &mockMetricsUsecase{
					DumpIteratorFunc: func(ctx context.Context) (
						func() (type_ string, name string, value string, exists bool), error,
					) {
						return nil, errors.New("some error")
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
			name: "mainpage: method not allowed",
			given: given{
				method:      http.MethodPost,
				url:         "/",
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
			assert.Equal(t, tt.want.callCount, len(tt.given.mockUsecase.calls.DumpIterator))
		})
	}
}
