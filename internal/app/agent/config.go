package agent

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v6"
)

const (
	defaultPollIntervalSec   = 2
	defaultReportIntervalSec = 10
	defaultServerAddress     = "localhost:8080"
)

type Config struct {
	PollIntervalSec   int    `env:"POLL_INTERVAL"`
	ReportIntervalSec int    `env:"REPORT_INTERVAL"`
	ServerAddress     string `env:"ADDRESS"`
}

func NewConfig() *Config {
	return &Config{
		PollIntervalSec:   defaultPollIntervalSec,
		ReportIntervalSec: defaultReportIntervalSec,
		ServerAddress:     defaultServerAddress,
	}
}

func (c *Config) ParseFlags() error {
	flag.IntVar(&c.PollIntervalSec, "p", c.PollIntervalSec,
		"interval between polling metrics, seconds; env: POLL_INTERVAL")
	flag.IntVar(&c.ReportIntervalSec, "r", c.ReportIntervalSec,
		"interval between sending metrics to server, seconds; env: REPORT_INTERVAL")
	flag.StringVar(&c.ServerAddress, "a", c.ServerAddress,
		"server address; env: ADDRESS")
	flag.CommandLine.Init("", flag.ContinueOnError)
	err := flag.CommandLine.Parse(os.Args[1:])
	if err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if flag.NArg() > 0 {
		flag.Usage()
		return errors.New("no positional arguments expected")
	}
	return nil
}

func (c *Config) ReadEnv() error {
	err := env.Parse(c)
	if err != nil {
		flag.Usage()
		return fmt.Errorf("read env: %w", err)
	}
	return nil
}

func ReadConfig() (*Config, error) {
	c := NewConfig()
	slog.Info("[main] default", "config", *c)
	// flags takes less priority according to task description
	err := c.ParseFlags()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after flags", "config", *c)
	// enviromnent takes higher priority according to task description
	err = c.ReadEnv()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after env", "config", *c)
	return c, nil
}
