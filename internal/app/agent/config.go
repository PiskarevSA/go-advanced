package agent

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v6"
)

const (
	defaultJSONConfigPath    = ""
	defaultPollIntervalSec   = 2
	defaultReportIntervalSec = 10
	defaultServerAddress     = "localhost:8080"
	defaultKey               = ""
	defaultRateLimit         = 1
	defaultCryptoKey         = ""
)

type Config struct {
	jsonConfigPath    string `env:"CONFIG"`
	PollIntervalSec   int    `env:"POLL_INTERVAL" json:"poll_interval"`
	ReportIntervalSec int    `env:"REPORT_INTERVAL" json:"report_interval"`
	ServerAddress     string `env:"ADDRESS" json:"address"`
	Key               string `env:"KEY" json:"key"`
	RateLimit         int    `env:"RATE_LIMIT" json:"rate_limit"`
	CryptoKey         string `env:"CRYPTO_KEY" json:"crypto_key"`
}

func NewConfig() *Config {
	result := &Config{
		jsonConfigPath:    defaultJSONConfigPath,
		PollIntervalSec:   defaultPollIntervalSec,
		ReportIntervalSec: defaultReportIntervalSec,
		ServerAddress:     defaultServerAddress,
		Key:               defaultKey,
		RateLimit:         defaultRateLimit,
		CryptoKey:         defaultCryptoKey,
	}
	flag.StringVar(&result.jsonConfigPath, "c", result.jsonConfigPath,
		"path to .json config file; env: CONFIG")
	flag.IntVar(&result.PollIntervalSec, "p", result.PollIntervalSec,
		"interval between polling metrics, seconds; env: POLL_INTERVAL")
	flag.IntVar(&result.ReportIntervalSec, "r", result.ReportIntervalSec,
		"interval between sending metrics to server, seconds; env: REPORT_INTERVAL")
	flag.StringVar(&result.ServerAddress, "a", result.ServerAddress,
		"server address; env: ADDRESS")
	flag.StringVar(&result.Key, "k", result.Key,
		"the key for signing the request body (the signature is in the HashSHA256 header); env: KEY")
	flag.IntVar(&result.RateLimit, "l", result.RateLimit,
		"max number of concurrent calls to server, flush to console if 0; env: RATE_LIMIT")
	flag.StringVar(&result.CryptoKey, "crypto-key", result.CryptoKey,
		"the path to the file with the server's public key for encrypting the message from the agent to the server; env: CRYPTO_KEY")
	return result
}

func (c Config) LogValue() slog.Value {
	// hide key
	if len(c.Key) > 0 {
		c.Key = "[redacted]"
	}
	return slog.GroupValue(
		slog.String("JSONConfigPath", c.jsonConfigPath),
		slog.Int("PollIntervalSec", c.PollIntervalSec),
		slog.Int("ReportIntervalSec", c.ReportIntervalSec),
		slog.String("ServerAddress", c.ServerAddress),
		slog.String("Key", c.Key),
		slog.Int("RateLimit", c.RateLimit),
		slog.String("CryptoKey", c.CryptoKey),
	)
}

func (c *Config) ParseFlags() error {
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

func (c *Config) JSONConfigPath() string {
	return c.jsonConfigPath
}

func (c *Config) ReadJSONFile() error {
	f, err := os.Open(c.jsonConfigPath)
	if err != nil {
		return fmt.Errorf("read json file: %w", err)
	}
	decoder := json.NewDecoder(f)
	err = decoder.Decode(c)
	if err != nil {
		return fmt.Errorf("read json file: %w", err)
	}
	return nil
}
