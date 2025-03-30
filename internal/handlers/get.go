package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/PiskarevSA/go-advanced/internal/handlers/validation"
)

const (
	GetAsJSONPattern = `/value/`
	GetAsTextPattern = `/value/{type}/{name}`
)

type GetterUsecase interface {
	GetMetric(metric entities.Metric) (*entities.Metric, error)
}

// GetAsJSONHandler handles endpoint: POST /value
// request type: "application/json", body: models.Metric
// response	type: "application/json", body: models.Metric
type GetAsJSONHandler struct {
	Getter GetterUsecase
}

func (h *GetAsJSONHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	validMetric, err := validation.ValidateMetricFromGetAsJSONRequest(req)
	if err != nil {
		handleGetterError(err, res, req)
		return
	}

	responseMetric, err := h.Getter.GetMetric(*validMetric)
	if err != nil {
		handleGetterError(err, res, req)
		return
	}

	// success
	response, err := validation.MakeResponseFromEntityMetric(*responseMetric)
	if err != nil {
		handleGetterError(err, res, req)
		return
	}
	res.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(res).Encode(response); err != nil {
		http.Error(res, err.Error(), http.StatusInternalServerError)
		return
	}
}

// GetAsTextHandler handles endpoint: GET /value/{type}/{name}
// request: none
// response type: "text/plain; charset=utf-8", body: metric value as string
type GetAsTextHandler struct {
	Getter GetterUsecase
}

func (h *GetAsTextHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	validMetric, err := validation.ValidateMetricFromGetGetAsTextRequest(req)
	if err != nil {
		handleUpdateError(err, res, req)
		return
	}

	responseMetric, err := h.Getter.GetMetric(*validMetric)
	if err != nil {
		handleGetterError(err, res, req)
		return
	}

	// success
	var response string
	switch responseMetric.Type {
	case entities.MetricTypeGauge:
		response = fmt.Sprint(responseMetric.Value)
	case entities.MetricTypeCounter:
		response = fmt.Sprint(responseMetric.Delta)
	default:
		err := entities.NewInternalError(
			"unexpected internal metric type: " + responseMetric.Type.String())
		handleGetterError(err, res, req)
		return
	}
	_, err = res.Write([]byte(response))
	if err != nil {
		err := entities.NewInternalError(
			"response writing error: " + responseMetric.Type.String())
		handleGetterError(err, res, req)
		return
	}
	res.Header().Set("Content-Type", "text/plain; charset=utf-8")
}

func handleGetterError(err error, res http.ResponseWriter, req *http.Request) {
	var (
		invalidMetricTypeError  *entities.InvalidMetricTypeError
		metricNameNotFoundError *entities.MetricNameNotFoundError
		jsonRequestDecodeError  *entities.JSONRequestDecodeError
		internalError           *entities.InternalError
	)
	switch {
	case errors.Is(err, entities.ErrJSONRequestExpected):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.Is(err, entities.ErrEmptyMetricName):
		http.NotFound(res, req)
	case errors.As(err, &invalidMetricTypeError):
		http.NotFound(res, req)
	case errors.As(err, &metricNameNotFoundError):
		http.NotFound(res, req)
	case errors.As(err, &jsonRequestDecodeError):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.As(err, &internalError):
		http.Error(res, err.Error(), http.StatusInternalServerError)
	default:
		// unexpected error
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
