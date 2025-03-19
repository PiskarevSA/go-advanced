package errors

import (
	"fmt"
)

// stateless errors
// .. EmptyMetricTypeError
type EmptyMetricTypeError struct{}

func NewEmptyMetricTypeError() *EmptyMetricTypeError {
	return &EmptyMetricTypeError{}
}

func (e EmptyMetricTypeError) Error() string {
	return "empty metric type"
}

// .. EmptyMetricNameError
type EmptyMetricNameError struct{}

func NewEmptyMetricNameError() *EmptyMetricNameError {
	return &EmptyMetricNameError{}
}

func (e *EmptyMetricNameError) Error() string {
	return "empty metric name"
}

// .. MissingValueError
type MissingValueError struct{}

func NewMissingValueError() *MissingValueError {
	return &MissingValueError{}
}

func (e *MissingValueError) Error() string {
	return "missing value"
}

// .. MissingDeltaError
type MissingDeltaError struct{}

func NewMissingDeltaError() *MissingDeltaError {
	return &MissingDeltaError{}
}

func (e *MissingDeltaError) Error() string {
	return "missing delta"
}

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
