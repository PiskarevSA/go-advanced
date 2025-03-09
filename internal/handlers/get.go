package handlers

import (
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/errors"
	"github.com/go-chi/chi/v5"
)

type Getter interface {
	Get(metricType string, metricName string) (
		value string, err error)
}

// GET /value/{type}/{name}
func Get(getter Getter) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		metricType := chi.URLParam(req, "type")
		metricName := chi.URLParam(req, "name")

		value, err := getter.Get(metricType, metricName)
		if err != nil {
			switch err.(type) {
			case *errors.EmptyMetricTypeError:
				http.NotFound(res, req)
			case *errors.InvalidMetricTypeError:
				http.NotFound(res, req)
			case *errors.MetricNameNotFoundError:
				http.NotFound(res, req)
			default:
				// unexpected error
				http.Error(res, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		// success
		_, err = res.Write([]byte(value))
		if err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
}
