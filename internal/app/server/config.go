package server

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/PiskarevSA/go-advanced/internal/logger"
	"github.com/caarlos0/env/v6"
)

type Config struct {
	ServerAddress string `env:"ADDRESS"`
}

const (
	defaultServerAddress = "localhost:8080"
)

func NewConfig() *Config {
	return &Config{
		ServerAddress: defaultServerAddress,
	}
}

func (c *Config) ParseFlags() error {
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
	logger.Sugar.Infof("default config: %+v\n", *c)
	// flags takes less priority according to task description
	err := c.ParseFlags()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	logger.Sugar.Infof("config after flags: %+v\n", *c)
	// enviromnent takes higher priority according to task description
	err = c.ReadEnv()
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	logger.Sugar.Infof("config after env: %+v\n", *c)
	return c, nil
}
