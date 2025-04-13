package adapters

import (
	"encoding/json"
	"fmt"
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

func ConvertMetricFromGetAsJSONRequest(req *http.Request) (*entities.Metric, error) {
	if req.Header.Get("Content-Type") != "application/json" {
		return nil, entities.ErrJSONRequestExpected
	}
	var metric models.Metric
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
		return nil, entities.NewJSONRequestDecodeError(err)
	}
	var result entities.Metric
	var err error
	result.Type, err = convertMetricType(metric.MType)
	if err != nil {
		return nil, err
	}
	result.Name, err = convertMetricName(metric.ID)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func ConvertMetricFromUpdateFromJSONRequest(req *http.Request) (*entities.Metric, error) {
	if req.Header.Get("Content-Type") != "application/json" {
		return nil, entities.ErrJSONRequestExpected
	}
	var metric models.Metric
	if err := json.NewDecoder(req.Body).Decode(&metric); err != nil {
		return nil, entities.NewJSONRequestDecodeError(err)
	}
	var result entities.Metric
	var err error
	result.Type, err = convertMetricType(metric.MType)
	if err != nil {
		return nil, err
	}
	result.Name, err = convertMetricName(metric.ID)
	if err != nil {
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
			"unexpected internal metric type: "+result.Type.String(), nil)
	}
	return &result, nil
}

func ConvertBatchMetricFromUpdateFromJSONRequest(req *http.Request) ([]entities.Metric, error) {
	if req.Header.Get("Content-Type") != "application/json" {
		return nil, entities.ErrJSONRequestExpected
	}
	var metrics []models.Metric
	if err := json.NewDecoder(req.Body).Decode(&metrics); err != nil {
		return nil, entities.NewJSONRequestDecodeError(err)
	}
	var result []entities.Metric
	for i, metric := range metrics {
		var entityMetric entities.Metric
		var err error
		entityMetric.Type, err = convertMetricType(metric.MType)
		if err != nil {
			return nil, fmt.Errorf("metric[%v]: %w", i, err)
		}
		entityMetric.Name, err = convertMetricName(metric.ID)
		if err != nil {
			return nil, fmt.Errorf("metric[%v]: %w", i, err)
		}
		switch entityMetric.Type {
		case entities.MetricTypeGauge:
			if metric.Value == nil {
				return nil, fmt.Errorf("metric[%v]: %w", i, entities.ErrMissingValue)
			}
			entityMetric.Value = entities.Gauge(*metric.Value)
		case entities.MetricTypeCounter:
			if metric.Delta == nil {
				return nil, fmt.Errorf("metric[%v]: %w", i, entities.ErrMissingDelta)
			}
			entityMetric.Delta = entities.Counter(*metric.Delta)
		default:
			return nil, entities.NewInternalError(
				fmt.Sprintf(
					"metric[%v]: unexpected internal metric type: %v",
					i, entityMetric.Type.String()), nil)
		}
		result = append(result, entityMetric)
	}
	return result, nil
}

func ConvertMetricFromGetGetAsTextRequest(req *http.Request) (*entities.Metric, error) {
	metricType := chi.URLParam(req, "type")
	metricName := chi.URLParam(req, "name")

	var result entities.Metric
	var err error
	result.Type, err = convertMetricType(metricType)
	if err != nil {
		return nil, err
	}
	result.Name, err = convertMetricName(metricName)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func ConvertMetricFromUpdateFromURLRequest(req *http.Request) (*entities.Metric, error) {
	metricType := chi.URLParam(req, "type")
	metricName := chi.URLParam(req, "name")
	metricValue := chi.URLParam(req, "value")

	var result entities.Metric
	var err error
	result.Type, err = convertMetricType(metricType)
	if err != nil {
		return nil, err
	}
	result.Name, err = convertMetricName(metricName)
	if err != nil {
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
			"unexpected internal metric type: "+result.Type.String(), nil)
	}
	return &result, nil
}

func ConvertEntityMetric(metric entities.Metric) (*models.Metric, error) {
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
			"unexpected internal metric type: "+metric.Type.String(), nil)
	}
	result.ID = string(metric.Name)
	return &result, nil
}

func ConvertEntityMetrics(metrics []entities.Metric) ([]models.Metric, error) {
	result := make([]models.Metric, 0)
	for i, entityMetric := range metrics {
		metric, err := ConvertEntityMetric(entityMetric)
		if err != nil {
			return nil, fmt.Errorf("metric[%v]: %w", i, err)
		}
		result = append(result, *metric)
	}
	return result, nil
}

func convertMetricType(metricType string) (entities.MetricType, error) {
	switch MetricType(metricType) {
	case MetricTypeGauge:
		return entities.MetricTypeGauge, nil
	case MetricTypeCounter:
		return entities.MetricTypeCounter, nil
	}
	return entities.MetricTypeUndefined, entities.NewInvalidMetricTypeError(metricType)
}

func convertMetricName(metricName string) (entities.MetricName, error) {
	if len(metricName) == 0 {
		return "", entities.ErrEmptyMetricName
	}
	return entities.MetricName(metricName), nil
}
