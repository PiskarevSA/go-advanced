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
	defaultJSONConfigPath    = ""
	defaultServerAddress     = "localhost:8080"
	defaultGrpcServerAddress = ":9090"
	defaultStoreInterval     = 300
	defaultFileStoragePath   = "metrics.json"
	defaultRestore           = false
	defaultDatabaseDSN       = ""
	defaultKey               = ""
	defaultCryptoKey         = ""
	defaultTrustedSubnet     = ""
)

type Config struct {
	jsonConfigPath    string `env:"CONFIG"`
	ServerAddress     string `env:"ADDRESS" json:"address"`
	GrpcServerAddress string `env:"GRPC_ADDRESS" json:"grpc_address"`
	StoreInterval     int    `env:"STORE_INTERVAL" json:"store_interval"`
	FileStoragePath   string `env:"FILE_STORAGE_PATH" json:"store_file"`
	Restore           bool   `env:"RESTORE" json:"restore"`
	DatabaseDSN       string `env:"DATABASE_DSN" json:"database_dsn"`
	Key               string `env:"KEY" json:"key"`
	CryptoKey         string `env:"CRYPTO_KEY" json:"crypto_key"`
	TrustedSubnet     string `env:"TRUSTED_SUBNET" json:"trusted_subnet"`
}

func NewConfig() *Config {
	result := &Config{
		jsonConfigPath:    defaultJSONConfigPath,
		ServerAddress:     defaultServerAddress,
		GrpcServerAddress: defaultGrpcServerAddress,
		StoreInterval:     defaultStoreInterval,
		FileStoragePath:   defaultFileStoragePath,
		Restore:           defaultRestore,
		DatabaseDSN:       defaultDatabaseDSN,
		Key:               defaultKey,
		CryptoKey:         defaultCryptoKey,
		TrustedSubnet:     defaultTrustedSubnet,
	}
	flag.StringVar(&result.jsonConfigPath, "c", result.jsonConfigPath,
		"path to .json config file; env: CONFIG")
	flag.StringVar(&result.ServerAddress, "a", result.ServerAddress,
		"server address; env: ADDRESS")
	flag.StringVar(&result.GrpcServerAddress, "g", result.GrpcServerAddress,
		"grpc server address; env: GRPC_ADDRESS")
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
	flag.StringVar(&result.TrustedSubnet, "t", result.TrustedSubnet,
		"trusted subnet as a string representation of classless inter-domain routing (CIDR); env: TRUSTED_SUBNET")
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
		slog.String("JSONConfigPath", c.jsonConfigPath),
		slog.String("ServerAddress", c.ServerAddress),
		slog.String("GrpcServerAddress", c.GrpcServerAddress),
		slog.Int("StoreInterval", c.StoreInterval),
		slog.String("FileStoragePath", c.FileStoragePath),
		slog.Bool("Restore", c.Restore),
		slog.String("DatabaseDSN", c.DatabaseDSN),
		slog.String("Key", c.Key),
		slog.String("CryptoKey", c.CryptoKey),
		slog.String("TrustedSubnet", c.TrustedSubnet),
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
