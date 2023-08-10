// Package logging provides a simple wrapper around slog to initialize the logger with the given configuration.
package logging

import (
	"fmt"
	"log/slog"
	"os"
)

var errInvalidLevel = fmt.Errorf("invalid log level, must be one of: debug, info, warn, error")
var errInvalidHandler = fmt.Errorf("invalid handler")
var errInvalidSettings = fmt.Errorf("invalid logging settings")

// Config defines the configuration for the logger.
type Config struct {
	Level   string `yaml:"level"`
	Handler string `yaml:"handler"`
}

func levelFromString(level string) (slog.Level, error) {
	switch level {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("%w. got %s", errInvalidLevel, level)
	}
}

func handlerFromString(handler string) (func(*slog.HandlerOptions) slog.Handler, error) {
	switch handler {
	case "text", "":
		return func(opts *slog.HandlerOptions) slog.Handler { return slog.NewTextHandler(os.Stdout, opts) }, nil
	case "json":
		return func(opts *slog.HandlerOptions) slog.Handler { return slog.NewJSONHandler(os.Stdout, opts) }, nil
	default:
		return nil, fmt.Errorf("%w: %s", errInvalidHandler, handler)
	}
}

// Initialize initializes the logger with the given logging configuration.
func (c *Config) Initialize() error {
	var opts slog.HandlerOptions
	level, err := levelFromString(c.Level)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidSettings, err)
	}
	opts.Level = level

	handler, err := handlerFromString(c.Handler)
	if err != nil {
		return fmt.Errorf("%w: %s", errInvalidSettings, err)
	}
	slog.SetDefault(slog.New(handler(&opts)))

	return nil
}
