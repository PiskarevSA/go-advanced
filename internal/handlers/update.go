package handlers

import (
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/errors"
	"github.com/go-chi/chi/v5"
)

type Updater interface {
	Update(metricType string, metricName string, metricValue string) error
}

// POST "text/plain" /update/{type}/{name}/{value}
func Update(updater Updater) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		// incorrect metric type should return http.StatusBadRequest
		metricType := chi.URLParam(req, "type")
		metricName := chi.URLParam(req, "name")
		metricValue := chi.URLParam(req, "value")

		err := updater.Update(metricType, metricName, metricValue)
		if err != nil {
			switch err.(type) {
			case *errors.EmptyMetricTypeError:
				http.Error(res, err.Error(), http.StatusBadRequest)
			case *errors.InvalidMetricTypeError:
				http.Error(res, err.Error(), http.StatusBadRequest)
			case *errors.EmptyMetricNameError:
				http.NotFound(res, req)
			case *errors.MetricValueIsNotValidError:
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			default:
				// unexpected error
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}
		}
		// success
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
}
