package storage

import (
	"testing"
)

func TestMemStorage_SetGauge(t *testing.T) {
	type args struct {
		key   string
		value float64
	}
	tests := []struct {
		name  string
		gauge map[string]float64
		args  args
		want  map[string]float64
	}{
		{
			name: "add gauge",
			gauge: map[string]float64{
				"foo": 1.0,
				"bar": 2.0,
			},
			args: args{
				key:   "baz",
				value: 3.0,
			},
			want: map[string]float64{
				"foo": 1.0,
				"bar": 2.0,
				"baz": 3.0,
			},
		},
		{
			name: "replace gauge",
			gauge: map[string]float64{
				"foo": 1.0,
				"bar": 2.0,
			},
			args: args{
				key:   "bar",
				value: 3.0,
			},
			want: map[string]float64{
				"foo": 1.0,
				"bar": 3.0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				gauge:   tt.gauge,
				counter: map[string]int64{},
			}
			m.SetGauge(tt.args.key, tt.args.value)
		})
	}
}

func TestMemStorage_IncreaseCounter(t *testing.T) {
	type args struct {
		key      string
		addition int64
	}
	tests := []struct {
		name    string
		counter map[string]int64
		args    args
		want    map[string]int64
	}{
		{
			name: "add counter",
			counter: map[string]int64{
				"foo": 1,
				"bar": 2,
			},
			args: args{
				key:      "baz",
				addition: 3,
			},
			want: map[string]int64{
				"foo": 1,
				"bar": 2,
				"baz": 3,
			},
		},
		{
			name: "increase counter",
			counter: map[string]int64{
				"foo": 1,
				"bar": 2,
			},
			args: args{
				key:      "bar",
				addition: 3,
			},
			want: map[string]int64{
				"foo": 1,
				"bar": 5,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MemStorage{
				gauge:   map[string]float64{},
				counter: tt.counter,
			}
			m.IncreaseCounter(tt.args.key, tt.args.addition)
		})
	}
}
