package entities

import (
	"errors"
	"fmt"
)

// stateless errors
var (
	ErrEmptyMetricName     = errors.New("empty metric name")
	ErrJSONRequestExpected = errors.New("expected Content-Type=application/json")
	ErrMissingValue        = errors.New("missing value")
	ErrMissingDelta        = errors.New("missing delta")
)

// stateful errors

// InvalidMetricTypeError returns with Bad Request or Not Found HTTP code
type InvalidMetricTypeError struct {
	MetricType string
}

func NewInvalidMetricTypeError(metricType string) error {
	return &InvalidMetricTypeError{MetricType: metricType}
}

func (e *InvalidMetricTypeError) Error() string {
	return fmt.Sprintf("invalid metric type: %s", e.MetricType)
}

// MetricNameNotFoundError returns with Not Found HTTP code
type MetricNameNotFoundError struct {
	MetricName MetricName
}

func NewMetricNameNotFoundError(metricName MetricName) error {
	return &MetricNameNotFoundError{MetricName: metricName}
}

func (e *MetricNameNotFoundError) Error() string {
	return fmt.Sprintf("metric name not found: %s", e.MetricName)
}

// MetricValueIsNotValidError returns with Bad Request HTTP code
type MetricValueIsNotValidError struct {
	error
}

func NewMetricValueIsNotValidError(error error) error {
	return &MetricValueIsNotValidError{error: error}
}

func (e *MetricValueIsNotValidError) Error() string {
	return fmt.Sprintf("invalid metric value: %s", e.error.Error())
}

func (e *MetricValueIsNotValidError) Unwrap() error {
	return e.error
}

// JSONRequestDecodeError returns with Bad Request HTTP code
type JSONRequestDecodeError struct {
	error
}

func NewJSONRequestDecodeError(error error) error {
	return &JSONRequestDecodeError{error: error}
}

func (e *JSONRequestDecodeError) Error() string {
	return fmt.Sprintf("json request decoding: %s", e.error.Error())
}

func (e *JSONRequestDecodeError) Unwrap() error {
	return e.error
}

// InternalError returns with Internal Server Error HTTP code
type InternalError struct {
	message string
	err     error
}

func NewInternalError(message string, err error) error {
	return &InternalError{
		message: message,
		err:     err,
	}
}

func (e *InternalError) Error() string {
	if e.err != nil {
		return fmt.Sprintf("internal error: %s: %v", e.message, e.err)
	} else {
		return fmt.Sprintf("internal error: %s", e.message)
	}
}

func (e *InternalError) Unwrap() error {
	return e.err
}
