package handlers

import (
	"fmt"
	"net/http"
	"strings"
)

// GET /value/{type}/{name}
func Get(repo Repositories) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		// incorrect or empty metric type should return http.NotFound
		tail := strings.TrimPrefix(req.URL.Path, "/value/")
		metricType, metricName, _ := strings.Cut(tail, "/")

		var str string
		switch metricType {
		case "gauge":
			value, exist := repo.Gauge(metricName)
			if !exist {
				http.NotFound(res, req)
				return
			}
			str = fmt.Sprint(value)

		case "counter":
			value, exist := repo.Counter(metricName)
			if !exist {
				http.NotFound(res, req)
				return
			}
			str = fmt.Sprint(value)
		default:
			http.NotFound(res, req)
			return
		}
		// success
		_, err := res.Write([]byte(str))
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
}
