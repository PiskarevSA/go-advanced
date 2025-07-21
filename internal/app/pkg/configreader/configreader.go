package configreader

import (
	"fmt"
	"log/slog"
)

type config interface {
	ParseFlags() error
	ReadEnv() error
	JSONConfigPath() string
	ReadJSONFile() error
}

// Do is a common pipeline for config reading
func Do(c config) error {
	slog.Info("[main] default", "config", c)
	// flags takes less priority according to task description
	err := c.ParseFlags()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after flags", "config", c)
	// environment takes higher priority according to task description
	err = c.ReadEnv()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after env", "config", c)

	// return if no json config file provided
	if len(c.JSONConfigPath()) == 0 {
		return nil
	}

	// json config file provided, but it have least priority, so we need
	// to read all configs again
	err = c.ReadJSONFile()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after json", "config", c)
	err = c.ParseFlags()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after flags repeated", "config", c)
	err = c.ReadEnv()
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}
	slog.Info("[main] after env repeated", "config", c)
	return nil
}
