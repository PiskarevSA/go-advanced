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
	defaultGrpcServerAddress = "127.0.0.1:9090"
	defaultWorkMode          = "rest"
	defaultKey               = ""
	defaultRateLimit         = 1
	defaultCryptoKey         = ""
)

type Config struct {
	jsonConfigPath    string `env:"CONFIG"`
	PollIntervalSec   int    `env:"POLL_INTERVAL" json:"poll_interval"`
	ReportIntervalSec int    `env:"REPORT_INTERVAL" json:"report_interval"`
	ServerAddress     string `env:"ADDRESS" json:"address"`
	GrpcServerAddress string `env:"GRPC_ADDRESS" json:"grpc_address"`
	WorkMode          string `env:"WORK_MODE" json:"work_mode"`
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
		GrpcServerAddress: defaultGrpcServerAddress,
		WorkMode:          defaultWorkMode,
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
	flag.StringVar(&result.GrpcServerAddress, "g", result.GrpcServerAddress,
		"grpc server address; env: GRPC_ADDRESS")
	flag.StringVar(&result.WorkMode, "m", result.WorkMode,
		"working mode: rest or grpc; env: MODE")
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
		slog.String("GrpcServerAddress", c.GrpcServerAddress),
		slog.String("WorkMode", c.WorkMode),
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
