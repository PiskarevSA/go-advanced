package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMemStorage_SetGauge(t *testing.T) {
	type given struct {
		gauge    map[string]float64
		argKey   string
		argValue float64
	}
	type want struct {
		gauge map[string]float64
	}
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "add gauge",
			given: given{
				gauge: map[string]float64{
					"foo": 1.0,
					"bar": 2.0,
				},
				argKey:   "baz",
				argValue: 3.0,
			},
			want: want{gauge: map[string]float64{
				"foo": 1.0,
				"bar": 2.0,
				"baz": 3.0,
			}},
		},
		{
			name: "replace gauge",
			given: given{
				gauge: map[string]float64{
					"foo": 1.0,
					"bar": 2.0,
				},
				argKey:   "bar",
				argValue: 3.0,
			},
			want: want{gauge: map[string]float64{
				"foo": 1.0,
				"bar": 3.0,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				gauge:   tt.given.gauge,
				counter: map[string]int64{},
			}
			m.SetGauge(tt.given.argKey, tt.given.argValue)
			assert.Equal(t, tt.want.gauge, m.gauge)
		})
	}
}

func TestMemStorage_Gauge(t *testing.T) {
	type given struct {
		gauge  map[string]float64
		argKey string
	}
	type want struct {
		argValue  float64
		argExists bool
	}
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "get not existed gauge",
			given: given{
				gauge: map[string]float64{
					"foo": 1.0,
					"bar": 2.0,
				},
				argKey: "baz",
			},
			want: want{
				argValue:  0,
				argExists: false,
			},
		},
		{
			name: "get existed gauge",
			given: given{
				gauge: map[string]float64{
					"foo": 1.0,
					"bar": 2.0,
				},
				argKey: "bar",
			},
			want: want{
				argValue:  2.0,
				argExists: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				gauge:   tt.given.gauge,
				counter: map[string]int64{},
			}
			value, exists := m.Gauge(tt.given.argKey)
			assert.Equal(t, tt.want.argValue, value)
			assert.Equal(t, tt.want.argExists, exists)
		})
	}
}

func TestMemStorage_IncreaseCounter(t *testing.T) {
	type given struct {
		counter  map[string]int64
		argKey   string
		argDelta int64
	}
	type want struct {
		counter  map[string]int64
		argValue int64
	}
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "add counter",
			given: given{
				counter: map[string]int64{
					"foo": 1,
					"bar": 2,
				},
				argKey:   "baz",
				argDelta: 3,
			},
			want: want{
				counter: map[string]int64{
					"foo": 1,
					"bar": 2,
					"baz": 3,
				},
				argValue: 3,
			},
		},
		{
			name: "increase counter",
			given: given{
				counter: map[string]int64{
					"foo": 1,
					"bar": 2,
				},
				argKey:   "bar",
				argDelta: 3,
			},
			want: want{
				counter: map[string]int64{
					"foo": 1,
					"bar": 5,
				},
				argValue: 5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				gauge:   map[string]float64{},
				counter: tt.given.counter,
			}
			value := m.IncreaseCounter(tt.given.argKey, tt.given.argDelta)
			assert.Equal(t, tt.want.counter, m.counter)
			assert.Equal(t, tt.want.argValue, value)
		})
	}
}

func TestMemStorage_Counter(t *testing.T) {
	type given struct {
		counter map[string]int64
		argKey  string
	}
	type want struct {
		argValue  int64
		argExists bool
	}
	tests := []struct {
		name  string
		given given
		want  want
	}{
		{
			name: "get not existed counter",
			given: given{
				counter: map[string]int64{
					"foo": 1,
					"bar": 2,
				},
				argKey: "baz",
			},
			want: want{
				argValue:  0,
				argExists: false,
			},
		},
		{
			name: "get existed counter",
			given: given{
				counter: map[string]int64{
					"foo": 1,
					"bar": 2,
				},
				argKey: "bar",
			},
			want: want{
				argValue:  2,
				argExists: true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				gauge:   map[string]float64{},
				counter: tt.given.counter,
			}
			value, exists := m.Counter(tt.given.argKey)
			assert.Equal(t, tt.want.argValue, value)
			assert.Equal(t, tt.want.argExists, exists)
		})
	}
}
