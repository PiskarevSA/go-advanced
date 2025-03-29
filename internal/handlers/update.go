package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/PiskarevSA/go-advanced/api"
	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/go-chi/chi/v5"
)

type Updater interface {
	Update(metricType string, metricName string, metricValue string) error
	IsGauge(metricType string) (bool, error)
	UpdateGauge(metricName string, value *float64) error
	IncreaseCounter(metricName string, delta *int64) (*int64, error)
}

// POST /update
// - req: "application/json", body: api.Metrics
// - res: "application/json", body: api.Metrics
func UpdateJSON(updater Updater) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "application/json" {
			http.Error(res, "expected Content-Type=application/json",
				http.StatusBadRequest)
			return
		}
		var metrics api.Metrics
		if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		metricType := metrics.MType
		metricName := metrics.ID

		isGauge, err := updater.IsGauge(metricType)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		if isGauge {
			if err := updater.UpdateGauge(metricName, metrics.Value); err != nil {
				handleUpdateError(err, res, req)
				return
			}
		} else {
			// overwrite metrics.Delta with accumulated value
			if metrics.Delta, err = updater.IncreaseCounter(metricName, metrics.Delta); err != nil {
				handleUpdateError(err, res, req)
				return
			}
		}
		// success
		res.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(res).Encode(&metrics); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// POST "text/plain" /update/{type}/{name}/{value}
// - req: body: none
// - res: "text/plain; charset=utf-8", body: none
func Update(updater Updater) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		metricType := chi.URLParam(req, "type")
		metricName := chi.URLParam(req, "name")
		metricValue := chi.URLParam(req, "value")

		if err := updater.Update(metricType, metricName, metricValue); err != nil {
			handleUpdateError(err, res, req)
			return
		}
		// success
		res.Header().Set("Content-Type", "text/plain; charset=utf-8")
	}
}

func handleUpdateError(err error, res http.ResponseWriter, req *http.Request) {
	var (
		invalidMetricTypeError     *entities.InvalidMetricTypeError
		metricValueIsNotValidError *entities.MetricValueIsNotValidError
	)
	// incorrect metric type should return http.StatusBadRequest
	switch {
	case errors.As(err, &invalidMetricTypeError):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.Is(err, entities.ErrEmptyMetricName):
		http.NotFound(res, req)
	case errors.As(err, &metricValueIsNotValidError):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.Is(err, entities.ErrMissingValue):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.Is(err, entities.ErrMissingDelta):
		http.Error(res, err.Error(), http.StatusBadRequest)
	default:
		// unexpected error
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
