package entities

import (
	"errors"
	"fmt"
)

// stateless errors
var (
	ErrEmptyMetricType = errors.New("empty metric type")
	ErrEmptyMetricName = errors.New("empty metric name")
	ErrMissingValue    = errors.New("missing value")
	ErrMissingDelta    = errors.New("missing delta")
)

// stateful errors
// .. InvalidMetricTypeError
type InvalidMetricTypeError struct {
	MetricType string
}

func NewInvalidMetricTypeError(metricType string) *InvalidMetricTypeError {
	return &InvalidMetricTypeError{MetricType: metricType}
}

func (e *InvalidMetricTypeError) Error() string {
	return fmt.Sprintf("invalid metric type: %s", e.MetricType)
}

// .. MetricNameNotFoundError
type MetricNameNotFoundError struct {
	MetricName string
}

func NewMetricNameNotFoundError(metricName string) *MetricNameNotFoundError {
	return &MetricNameNotFoundError{MetricName: metricName}
}

func (e *MetricNameNotFoundError) Error() string {
	return fmt.Sprintf("metric name not found: %s", e.MetricName)
}

// .. MetricValueIsNotValidError
type MetricValueIsNotValidError struct {
	error
}

func NewMetricValueIsNotValidError(error error) *MetricValueIsNotValidError {
	return &MetricValueIsNotValidError{error: error}
}

func (e *MetricValueIsNotValidError) Error() string {
	return fmt.Sprintf("invalid metric value: %s", e.error.Error())
}

func (e *MetricValueIsNotValidError) Unwrap() error {
	return e.error
}
