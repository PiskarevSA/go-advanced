package handlers

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

// POST "text/plain" /update/{type}/{name}/{value}
func Update(repo Repositories) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		// TODO PR #5
		// Вот эти все параметры в каждом хэндлере можно валидировать и если они
		// неправильные, возвращать не NotFound, а BadRequest

		// incorrect metric type should return http.StatusBadRequest
		metricType := chi.URLParam(req, "type")
		metricName := chi.URLParam(req, "name")
		metricValue := chi.URLParam(req, "value")

		switch metricType {
		case "gauge":
			if len(metricName) == 0 {
				http.NotFound(res, req)
				return
			}
			f64, err := strconv.ParseFloat(metricValue, 64)
			if err != nil {
				http.Error(res, "incorrect metric value",
					http.StatusBadRequest)
				return
			}
			repo.SetGauge(metricName, f64)
		case "counter":
			if len(metricName) == 0 {
				http.NotFound(res, req)
				return
			}
			i64, err := strconv.ParseInt(metricValue, 10, 64)
			if err != nil {
				http.Error(res, "incorrect metric value",
					http.StatusBadRequest)
				return
			}
			repo.SetCounter(metricName, i64)
		case "":
			http.Error(res, "empty metric type",
				http.StatusBadRequest)
			return
		default:
			http.Error(res, "unexpected metric type",
				http.StatusBadRequest)
			return
		}
		// success
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
}
