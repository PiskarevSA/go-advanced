package handlers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/PiskarevSA/go-advanced/internal/handlers/validation"
)

type Getter interface {
	Get(metric entities.Metric) (*entities.Metric, error)
}

// POST /value
// - req: "application/json", body: models.Metric
// - res: "application/json", body: models.Metric
func GetAsJSON(getter Getter) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		validMetric, err := validation.ValidateMetricFromGetAsJSONRequest(req)
		if err != nil {
			handleGetterError(err, res, req)
			return
		}

		responseMetric, err := getter.Get(*validMetric)
		if err != nil {
			handleGetterError(err, res, req)
			return
		}

		// success
		response := validation.MakeResponseFromEntityMetric(*responseMetric)
		res.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(res).Encode(response); err != nil {
			http.Error(res, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

// GET /value/{type}/{name}
// - res: "text/plain; charset=utf-8", body: metric value as string
func GetAsText(getter Getter) func(res http.ResponseWriter, req *http.Request) {
	return func(res http.ResponseWriter, req *http.Request) {
		validMetric, err := validation.ValidateMetricFromGetGetAsTextRequest(req)
		if err != nil {
			handleUpdateError(err, res, req)
			return
		}

		responseMetric, err := getter.Get(*validMetric)
		if err != nil {
			handleGetterError(err, res, req)
			return
		}

		// success
		var response string
		if responseMetric.IsGauge {
			response = fmt.Sprint(responseMetric.Value)
		} else {
			response = fmt.Sprint(responseMetric.Delta)
		}
		_, err = res.Write([]byte(response))
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
		jsonRequestDecodeError  *entities.JsonRequestDecodeError
	)
	switch {
	case errors.Is(err, entities.ErrJsonRequestExpected):
		http.Error(res, err.Error(), http.StatusBadRequest)
	case errors.Is(err, entities.ErrEmptyMetricName):
		http.NotFound(res, req)
	case errors.As(err, &invalidMetricTypeError):
		http.NotFound(res, req)
	case errors.As(err, &metricNameNotFoundError):
		http.NotFound(res, req)
	case errors.As(err, &jsonRequestDecodeError):
		http.Error(res, err.Error(), http.StatusBadRequest)
	default:
		// unexpected error
		http.Error(res, err.Error(), http.StatusInternalServerError)
	}
}
