package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/PiskarevSA/go-advanced/internal/models"
	"github.com/go-chi/chi/v5"
)

type Updater interface {
	UpdateMetric(type_, name, value string) error
	IsGauge(type_ string) (bool, error)
	UpdateGauge(name string, value *float64) error
	IncreaseCounter(name string, delta *int64) (*int64, error)
}

// POST /update
// - req: "application/json", body: models.Metric
// - res: "application/json", body: models.Metric
func UpdateFromJSON(updater Updater) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "application/json" {
			http.Error(res, "expected Content-Type=application/json",
				http.StatusBadRequest)
			return
		}
		var metric models.Metric
		if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}
		metricType := metric.MType
		metricName := metric.ID

		isGauge, err := updater.IsGauge(metricType)
		if err != nil {
			http.Error(res, err.Error(), http.StatusBadRequest)
			return
		}

		if isGauge {
			if err := updater.UpdateGauge(metricName, metric.Value); err != nil {
				handleUpdateError(err, res, req)
				return
			}
		} else {
			// overwrite metric.Delta with accumulated value
			if metric.Delta, err = updater.IncreaseCounter(metricName, metric.Delta); err != nil {
				handleUpdateError(err, res, req)
				return
			}
		}
		// success
		res.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(res).Encode(&metric); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// POST "text/plain" /update/{type}/{name}/{value}
// - req: body: none
// - res: "text/plain; charset=utf-8", body: none
func UpdateFromURL(updater Updater) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		metricType := chi.URLParam(req, "type")
		metricName := chi.URLParam(req, "name")
		metricValue := chi.URLParam(req, "value")

		if err := updater.UpdateMetric(metricType, metricName, metricValue); err != nil {
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
