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
	Get(metricType string, metricName string) (value string, err error)
	IsGauge(metricType string) (bool, error)
	GetGauge(metricName string) (value *float64, err error)
	GetCounter(metricName string) (value *int64, err error)
}

// POST /value
// - req: "application/json", body: api.Metrics
// - res: "application/json", body: api.Metrics
func GetJSON(getter Getter) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "application/json" {
			http.Error(res, "expected Content-Type=application/json",
				http.StatusBadRequest)
		}
		var metrics api.Metrics
		if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		metricType := metrics.MType
		metricName := metrics.ID

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
			metrics.Value = gauge
			metrics.Delta = nil
		} else {
			counter, err := getter.GetCounter(metricName)
			if err != nil {
				handleGetterError(err, res, req)
				return
			}

			metrics.Delta = counter
			metrics.Value = nil
		}

		// success
		res.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(res).Encode(&metrics); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// GET /value/{type}/{name}
// - res: "text/plain; charset=utf-8", body: metric value as string
func Get(getter Getter) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		metricType := chi.URLParam(req, "type")
		metricName := chi.URLParam(req, "name")

		value, err := getter.Get(metricType, metricName)
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
	case errors.Is(err, entities.ErrEmptyMetricType):
		http.NotFound(res, req)
	case errors.As(err, &invalidMetricTypeError):
		http.NotFound(res, req)
	case errors.As(err, &metricNameNotFoundError):
		http.NotFound(res, req)
	default:
		// unexpected error
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
