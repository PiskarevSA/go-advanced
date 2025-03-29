package entities

type (
	MetricName string
	Gauge      float64
	Counter    int64
)

type Metric struct {
	IsGauge bool
	Name    MetricName
	Value   Gauge   // актуально, когда и если IsGauge = true
	Delta   Counter // актуально, когда и если IsGauge = false
}
