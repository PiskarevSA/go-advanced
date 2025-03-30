package validation

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/PiskarevSA/go-advanced/internal/entities"
	"github.com/PiskarevSA/go-advanced/internal/models"
	"github.com/go-chi/chi/v5"
)

type (
	MetricType string
)

const (
	MetricTypeGauge   MetricType = "gauge"
	MetricTypeCounter MetricType = "counter"
)

func ValidateMetricFromGetAsJSONRequest(req *http.Request) (*entities.Metric, error) {
	if req.Header.Get("Content-Type") != "application/json" {
		return nil, entities.ErrJSONRequestExpected
	}
	var metric models.Metric
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
		return nil, entities.NewJSONRequestDecodeError(err)
	}
	var result entities.Metric
	if err := validateMetricType(&result, metric.MType); err != nil {
		return nil, err
	}
	if err := validateMetricName(&result, metric.ID); err != nil {
		return nil, err
	}
	return &result, nil
}

func ValidateMetricFromUpdateFromJSONRequest(req *http.Request) (*entities.Metric, error) {
	if req.Header.Get("Content-Type") != "application/json" {
		return nil, entities.ErrJSONRequestExpected
	}
	var metric models.Metric
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
		return nil, entities.NewJSONRequestDecodeError(err)
	}
	var result entities.Metric
	if err := validateMetricType(&result, metric.MType); err != nil {
		return nil, err
	}
	if err := validateMetricName(&result, metric.ID); err != nil {
		return nil, err
	}
	switch result.Type {
	case entities.MetricTypeGauge:
		if metric.Value == nil {
			return nil, entities.ErrMissingValue
		}
		result.Value = entities.Gauge(*metric.Value)
	case entities.MetricTypeCounter:
		if metric.Delta == nil {
			return nil, entities.ErrMissingDelta
		}
		result.Delta = entities.Counter(*metric.Delta)
	default:
		return nil, entities.NewInternalError(
			"unexpected internal metric type: " + result.Type.String())
	}
	return &result, nil
}

func ValidateMetricFromGetGetAsTextRequest(req *http.Request) (*entities.Metric, error) {
	metricType := chi.URLParam(req, "type")
	metricName := chi.URLParam(req, "name")

	var result entities.Metric
	if err := validateMetricType(&result, metricType); err != nil {
		return nil, err
	}
	if err := validateMetricName(&result, metricName); err != nil {
		return nil, err
	}
	return &result, nil
}

func ValidateMetricFromUpdateFromURLRequest(req *http.Request) (*entities.Metric, error) {
	metricType := chi.URLParam(req, "type")
	metricName := chi.URLParam(req, "name")
	metricValue := chi.URLParam(req, "value")

	var result entities.Metric
	if err := validateMetricType(&result, metricType); err != nil {
		return nil, err
	}
	if err := validateMetricName(&result, metricName); err != nil {
		return nil, err
	}
	if len(metricValue) == 0 {
		return nil, entities.ErrMissingValue
	}
	switch result.Type {
	case entities.MetricTypeGauge:
		asFloat64, err := strconv.ParseFloat(metricValue, 64)
		if err != nil {
			return nil, entities.NewMetricValueIsNotValidError(err)
		}

		result.Value = entities.Gauge(asFloat64)
	case entities.MetricTypeCounter:
		asInt64, err := strconv.ParseInt(metricValue, 10, 64)
		if err != nil {
			return nil, entities.NewMetricValueIsNotValidError(err)
		}
		result.Delta = entities.Counter(asInt64)
	default:
		return nil, entities.NewInternalError(
			"unexpected internal metric type: " + result.Type.String())
	}
	return &result, nil
}

func MakeResponseFromEntityMetric(metric entities.Metric) (*models.Metric, error) {
	var result models.Metric
	switch metric.Type {
	case entities.MetricTypeGauge:
		result.MType = string(MetricTypeGauge)
		result.Value = (*float64)(&metric.Value)
	case entities.MetricTypeCounter:
		result.MType = string(MetricTypeCounter)
		result.Delta = (*int64)(&metric.Delta)
	default:
		return nil, entities.NewInternalError(
			"unexpected internal metric type: " + metric.Type.String())
	}
	result.ID = string(metric.Name)
	return &result, nil
}

func validateMetricType(m *entities.Metric, metricType string) error {
	switch MetricType(metricType) {
	case MetricTypeGauge:
		m.Type = entities.MetricTypeGauge
		return nil
	case MetricTypeCounter:
		m.Type = entities.MetricTypeCounter
		return nil
	}
	return entities.NewInvalidMetricTypeError(metricType)
}

func validateMetricName(m *entities.Metric, metricName string) error {
	if len(metricName) == 0 {
		return entities.ErrEmptyMetricName
	}
	m.Name = entities.MetricName(metricName)
	return nil
}
