package entities

import (
	"errors"
	"fmt"
)

// stateless errors
var (
	ErrEmptyMetricName     = errors.New("empty metric name")
	ErrJsonRequestExpected = errors.New("expected Content-Type=application/json")
	ErrMissingValue        = errors.New("missing value")
	ErrMissingDelta        = errors.New("missing delta")
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
	MetricName MetricName
}

func NewMetricNameNotFoundError(metricName MetricName) *MetricNameNotFoundError {
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

// .. JsonRequestDecodeError
type JsonRequestDecodeError struct {
	error
}

func NewJsonRequestDecodeError(error error) *JsonRequestDecodeError {
	return &JsonRequestDecodeError{error: error}
}

func (e *JsonRequestDecodeError) Error() string {
	return fmt.Sprintf("json request decoding: %s", e.error.Error())
}

func (e *JsonRequestDecodeError) Unwrap() error {
	return e.error
}
