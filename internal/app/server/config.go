package server

import (
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
	defaultServerAddress   = "localhost:8080"
	defaultStoreInterval   = 300
	defaultFileStoragePath = "metrics.json"
	defaultRestore         = false
	defaultDatabaseDSN     = ""
	defaultKey             = ""
	defaultCryptoKey       = ""
)

type Config struct {
	ServerAddress   string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
	Key             string `env:"KEY"`
	CryptoKey       string `env:"CRYPTO_KEY"`
}

func NewConfig() *Config {
	return &Config{
		ServerAddress:   defaultServerAddress,
		StoreInterval:   defaultStoreInterval,
		FileStoragePath: defaultFileStoragePath,
		Restore:         defaultRestore,
		DatabaseDSN:     defaultDatabaseDSN,
		Key:             defaultKey,
		CryptoKey:       defaultCryptoKey,
	}
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
	flag.StringVar(&c.ServerAddress, "a", c.ServerAddress,
		"server address; env: ADDRESS")
	flag.IntVar(&c.StoreInterval, "i", c.StoreInterval,
		"metrics store inverval in seconds; env: STORE_INTERVAL")
	flag.StringVar(&c.FileStoragePath, "f", c.FileStoragePath,
		"path to file with stored metrics; env: FILE_STORAGE_PATH")
	flag.BoolVar(&c.Restore, "r", c.Restore,
		"restore metrics from file on service startup")
	flag.StringVar(&c.DatabaseDSN, "d", c.DatabaseDSN,
		"database data source name (DSN)")
	flag.StringVar(&c.Key, "k", c.Key,
		"the key for validating the request body and signing the response body (both signatures are in the HashSHA256 header); env: KEY")
	flag.StringVar(&c.CryptoKey, "crypto-key", c.CryptoKey,
		"The path to the file with the server's private key for decrypting the message from the agent to the server; env: CRYPTO_KEY")
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
	// environment takes higher priority according to task description
	err = c.ReadEnv()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after env", "config", *c)
	return c, nil
}
