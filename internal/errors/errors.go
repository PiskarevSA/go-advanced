package errors

import (
	"fmt"
)

// stateless errors
type ErrEmptyMetricName struct{}

func (e *ErrEmptyMetricName) Error() string {
	return "empty metric name"
}

// InvalidMetricTypeError
type InvalidMetricTypeError struct {
	MetricType string
}

func NewInvalidMetricTypeError(metricType string) *InvalidMetricTypeError {
	return &InvalidMetricTypeError{MetricType: metricType}
}

func (e *InvalidMetricTypeError) Error() string {
	return fmt.Sprintf("invalid metric type: %s", e.MetricType)
}

// MetricNameNotFoundError
type MetricNameNotFoundError struct {
	MetricName string
}

func NewMetricNameNotFoundError(metricName string) *MetricNameNotFoundError {
	return &MetricNameNotFoundError{MetricName: metricName}
}

func (e *MetricNameNotFoundError) Error() string {
	return fmt.Sprintf("metric name not found: %s", e.MetricName)
}
