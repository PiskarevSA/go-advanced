package server

import (
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddress   string `env:"ADDRESS"`
	StoreInterval   int    `env:"STORE_INTERVAL"`
	FileStoragePath string `env:"FILE_STORAGE_PATH"`
	Restore         bool   `env:"RESTORE"`
	DatabaseDSN     string `env:"DATABASE_DSN"`
}

const (
	defaultServerAddress   = "localhost:8080"
	defaultStoreInterval   = 300
	defaultFileStoragePath = "metrics.json"
	defaultRestore         = false
	defaultDatabaseDSN     = ""
)

func NewConfig() *Config {
	return &Config{
		ServerAddress:   defaultServerAddress,
		StoreInterval:   defaultStoreInterval,
		FileStoragePath: defaultFileStoragePath,
		Restore:         defaultRestore,
		DatabaseDSN:     defaultDatabaseDSN,
	}
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
