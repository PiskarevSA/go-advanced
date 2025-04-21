package metrics

import (
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/mem"
)

func PollGopsutilMetrics(gaugeMap map[string]Gauge, counter map[string]Counter) {
	v, err := mem.VirtualMemory()
	if err != nil {
		gaugeMap["TotalMemory"] = Gauge(v.Total)
		gaugeMap["FreeMemory"] = Gauge(v.Free)
	}
	c, err := cpu.Percent(0, false)
	if err != nil {
		gaugeMap["CPUutilization1"] = Gauge(c[0])
	}
}
