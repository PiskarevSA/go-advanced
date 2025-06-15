package entities

import (
	"fmt"
	"strconv"
)

type (
	MetricType int
	MetricName string
	Gauge      float64
	Counter    int64
)

// MetricType enumerator
const (
	MetricTypeUndefined MetricType = iota
	MetricTypeGauge
	MetricTypeCounter
)

func (t MetricType) String() string {
	var asStr string
	switch t {
	case MetricTypeUndefined:
		asStr = "undefined"
	case MetricTypeGauge:
		asStr = "gauge"
	case MetricTypeCounter:
		asStr = "counter"
	default:
		asStr = "unexpected"
	}
	return fmt.Sprintf("%v (i.e. %v)", strconv.Itoa(int(t)), asStr)
}

// Metric entity used by business logic (usecases)
type Metric struct {
	Type MetricType
	Name MetricName
	// .. данные MetricTypeGauge
	Value Gauge
	// .. данные MetricTypeCounter
	Delta Counter
}
