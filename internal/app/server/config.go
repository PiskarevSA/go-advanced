package server

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"strings"

	"github.com/caarlos0/env/v6"
)

const (
	defaultJSONConfigPath  = ""
	defaultServerAddress   = "localhost:8080"
	defaultStoreInterval   = 300
	defaultFileStoragePath = "metrics.json"
	defaultRestore         = false
	defaultDatabaseDSN     = ""
	defaultKey             = ""
	defaultCryptoKey       = ""
)

type Config struct {
	JSONConfigPath  string `env:"CONFIG"`
	ServerAddress   string `env:"ADDRESS" json:"address"`
	StoreInterval   int    `env:"STORE_INTERVAL" json:"store_interval"`
	FileStoragePath string `env:"FILE_STORAGE_PATH" json:"store_file"`
	Restore         bool   `env:"RESTORE" json:"restore"`
	DatabaseDSN     string `env:"DATABASE_DSN" json:"database_dsn"`
	Key             string `env:"KEY" json:"key"`
	CryptoKey       string `env:"CRYPTO_KEY" json:"crypto_key"`
}

func NewConfig() *Config {
	result := &Config{
		JSONConfigPath:  defaultJSONConfigPath,
		ServerAddress:   defaultServerAddress,
		StoreInterval:   defaultStoreInterval,
		FileStoragePath: defaultFileStoragePath,
		Restore:         defaultRestore,
		DatabaseDSN:     defaultDatabaseDSN,
		Key:             defaultKey,
		CryptoKey:       defaultCryptoKey,
	}
	flag.StringVar(&result.JSONConfigPath, "c", result.JSONConfigPath,
		"path to .json config file; env: CONFIG")
	flag.StringVar(&result.ServerAddress, "a", result.ServerAddress,
		"server address; env: ADDRESS")
	flag.IntVar(&result.StoreInterval, "i", result.StoreInterval,
		"metrics store inverval in seconds; env: STORE_INTERVAL")
	flag.StringVar(&result.FileStoragePath, "f", result.FileStoragePath,
		"path to file with stored metrics; env: FILE_STORAGE_PATH")
	flag.BoolVar(&result.Restore, "r", result.Restore,
		"restore metrics from file on service startup")
	flag.StringVar(&result.DatabaseDSN, "d", result.DatabaseDSN,
		"database data source name (DSN)")
	flag.StringVar(&result.Key, "k", result.Key,
		"the key for validating the request body and signing the response body (both signatures are in the HashSHA256 header); env: KEY")
	flag.StringVar(&result.CryptoKey, "crypto-key", result.CryptoKey,
		"The path to the file with the server's private key for decrypting the message from the agent to the server; env: CRYPTO_KEY")
	return result
}

func (c Config) LogValue() slog.Value {
	// hide database password
	re := regexp.MustCompile(`(?i)password=([^\s]+)`)
	match := re.FindStringSubmatch(c.DatabaseDSN)
	if len(match) > 1 {
		c.DatabaseDSN = strings.Replace(c.DatabaseDSN, match[1], "[redacted]", -1)
	}
	// hide key
	if len(c.Key) > 0 {
		c.Key = "[redacted]"
	}

	return slog.GroupValue(
		slog.String("JSONConfigPath", c.JSONConfigPath),
		slog.String("ServerAddress", c.ServerAddress),
		slog.Int("StoreInterval", c.StoreInterval),
		slog.String("FileStoragePath", c.FileStoragePath),
		slog.Bool("Restore", c.Restore),
		slog.String("DatabaseDSN", c.DatabaseDSN),
		slog.String("Key", c.Key),
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

func (c *Config) ReadJSONFile() error {
	f, err := os.Open(c.JSONConfigPath)
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

func ReadConfig() (*Config, error) {
	c := NewConfig()
	slog.Info("[main] default", "config", *c)
	// flags takes less priority according to task description
	err := c.ParseFlags()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after flags", "config", *c)
	// environment takes higher priority according to task description
	err = c.ReadEnv()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after env", "config", *c)

	// return if no json config file provided
	if len(c.JSONConfigPath) == 0 {
		return c, nil
	}

	// json config file provided, but it have least priority, so we need
	// to read all configs again
	err = c.ReadJSONFile()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after json", "config", *c)
	err = c.ParseFlags()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after flags repeated", "config", *c)
	err = c.ReadEnv()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after env repeated", "config", *c)
	return c, nil
}
