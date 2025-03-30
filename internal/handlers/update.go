package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/PiskarevSA/go-advanced/internal/handlers/validation"
)

const (
	UpdateFromJSONPattern = `/update/`
	UpdateFromURLPattern  = `/update/{type}/{name}/{value}`
)

type UpdaterUsecase interface {
	UpdateMetric(metric entities.Metric) (*entities.Metric, error)
}

// UpdateFromJSONHandler handles endpoint: POST /update
// request type: "application/json", body: models.Metric
// response	type: "application/json", body: models.Metric
type UpdateFromJSONHandler struct {
	Updater UpdaterUsecase
}

func (h *UpdateFromJSONHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	validMetric, err := validation.ValidateMetricFromUpdateFromJSONRequest(req)
	if err != nil {
		handleUpdateError(err, res, req)
		return
	}

	updatedMetric, err := h.Updater.UpdateMetric(*validMetric)
	if err != nil {
		handleUpdateError(err, res, req)
		return
	}
	// success
	response, err := validation.MakeResponseFromEntityMetric(*updatedMetric)
	if err != nil {
		handleUpdateError(err, res, req)
	}
	res.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(res).Encode(response); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

// UpdateFromURLHandler handles endpoint: POST /update/{type}/{name}/{value}
// request: none
// response	type: "text/plain; charset=utf-8", body: none
type UpdateFromURLHandler struct {
	Updater UpdaterUsecase
}

func (h *UpdateFromURLHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	validMetric, err := validation.ValidateMetricFromUpdateFromURLRequest(req)
	if err != nil {
		handleUpdateError(err, res, req)
		return
	}

	if _, err := h.Updater.UpdateMetric(*validMetric); err != nil {
		handleUpdateError(err, res, req)
		return
	}
	// success
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
}

func handleUpdateError(err error, res http.ResponseWriter, req *http.Request) {
	var (
		invalidMetricTypeError     *entities.InvalidMetricTypeError
		metricValueIsNotValidError *entities.MetricValueIsNotValidError
		jsonRequestDecodeError     *entities.JSONRequestDecodeError
		internalError              *entities.InternalError
	)
	// incorrect metric type should return http.StatusBadRequest
	switch {
	case errors.Is(err, entities.ErrJSONRequestExpected):
		http.Error(res, err.Error(), http.StatusBadRequest)
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
	case errors.As(err, &jsonRequestDecodeError):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.As(err, &internalError):
		http.Error(res, err.Error(), http.StatusInternalServerError)
	default:
		// unexpected error
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
