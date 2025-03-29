package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/PiskarevSA/go-advanced/api"
	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/go-chi/chi/v5"
)

type Getter interface {
	GetMetric(type_, name string) (value string, err error)
	IsGauge(type_ string) (bool, error)
	GetGauge(name string) (value *float64, err error)
	GetCounter(name string) (value *int64, err error)
}

// POST /value
// - req: "application/json", body: api.Metric
// - res: "application/json", body: api.Metric
func GetAsJSON(getter Getter) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "application/json" {
			http.Error(res, "expected Content-Type=application/json",
				http.StatusBadRequest)
		}
		var metric api.Metric
		if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		metricType := metric.MType
		metricName := metric.ID

		isGauge, err := getter.IsGauge(metricType)
		if err != nil {
			handleGetterError(err, res, req)
			return
		}

		if isGauge {
			gauge, err := getter.GetGauge(metricName)
			if err != nil {
				handleGetterError(err, res, req)
				return
			}
			metric.Value = gauge
			metric.Delta = nil
		} else {
			counter, err := getter.GetCounter(metricName)
			if err != nil {
				handleGetterError(err, res, req)
				return
			}

			metric.Delta = counter
			metric.Value = nil
		}

		// success
		res.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(res).Encode(&metric); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// GET /value/{type}/{name}
// - res: "text/plain; charset=utf-8", body: metric value as string
func GetAsText(getter Getter) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		metricType := chi.URLParam(req, "type")
		metricName := chi.URLParam(req, "name")

		value, err := getter.GetMetric(metricType, metricName)
		if err != nil {
			handleGetterError(err, res, req)
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

func handleGetterError(err error, res http.ResponseWriter, req *http.Request) {
	var (
		invalidMetricTypeError  *entities.InvalidMetricTypeError
		metricNameNotFoundError *entities.MetricNameNotFoundError
	)
	switch {
	case errors.As(err, &invalidMetricTypeError):
		http.NotFound(res, req)
	case errors.As(err, &metricNameNotFoundError):
		http.NotFound(res, req)
	default:
		// unexpected error
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
