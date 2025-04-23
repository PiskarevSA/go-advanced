package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_metrics_Poll(t *testing.T) {
	pollFunc := func(gauge map[string]Gauge, counter map[string]Counter) {
		gauge["foo"] += 1.234
		counter["bar"] += 456
	}

	m := NewPoller(pollFunc)
	m.Poll()
	assert.Equal(t, Gauge(1.234), m.metrics.Gauge["foo"])
	assert.Equal(t, Counter(456), m.metrics.Counter["bar"])
	m.Poll()
	assert.Equal(t, Gauge(2.468), m.metrics.Gauge["foo"])
	assert.Equal(t, Counter(912), m.metrics.Counter["bar"])
}

func Test_metrics_Read(t *testing.T) {
	pollFunc := func(gauge map[string]Gauge, counter map[string]Counter) {
		gauge["foo"] += 1.234
		counter["bar"] += 456
	}
	m := NewPoller(pollFunc)
	m.Poll()
	m.Poll()
	pollCount, g, c := m.Get()
	assert.Equal(t, 2, pollCount)
	assert.GreaterOrEqual(t, Gauge(2.468), g["foo"])
	assert.Equal(t, Counter(912), c["bar"])
}
