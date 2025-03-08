package agent

type Config struct {
	PollIntervalSec   int    `env:"POLL_INTERVAL"`
	ReportIntervalSec int    `env:"REPORT_INTERVAL"`
	ServerAddress     string `env:"ADDRESS"`
}
