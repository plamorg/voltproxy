package logging

import (
	"errors"
	"log/slog"
	"os"
	"reflect"
	"testing"
)

func TestLevelFromStringSuccess(t *testing.T) {
	tests := []struct {
		input         string
		expectedLevel slog.Level
	}{
		{
			input:         "debug",
			expectedLevel: slog.LevelDebug,
		},
		{
			input:         "info",
			expectedLevel: slog.LevelInfo,
		},
		{
			input:         "warn",
			expectedLevel: slog.LevelWarn,
		},
		{
			input:         "error",
			expectedLevel: slog.LevelError,
		},
		{
			input:         "",
			expectedLevel: slog.LevelInfo,
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			level, err := levelFromString(test.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if level != test.expectedLevel {
				t.Fatalf("expected level %s, got %s", test.expectedLevel, level)
			}
		})
	}
}

func TestLevelFromStringError(t *testing.T) {
	_, err := levelFromString("invalid")
	if !errors.Is(err, errInvalidLevel) {
		t.Fatalf("expected error %s, got %s", errInvalidLevel, err)
	}
}

func TestHandlerFromStringSuccess(t *testing.T) {
	opts := slog.HandlerOptions{
		Level: slog.LevelWarn,
	}

	tests := []struct {
		input           string
		expectedHandler slog.Handler
	}{
		{
			input:           "text",
			expectedHandler: slog.NewTextHandler(os.Stdout, &opts),
		},
		{
			input:           "",
			expectedHandler: slog.NewTextHandler(os.Stdout, &opts),
		},
		{
			input:           "json",
			expectedHandler: slog.NewJSONHandler(os.Stdout, &opts),
		},
	}

	for _, test := range tests {
		t.Run(test.input, func(t *testing.T) {
			handler, err := handlerFromString(test.input)
			if err != nil {
				t.Fatalf("unexpected error: %s", err)
			}

			if !reflect.DeepEqual(handler(&opts), test.expectedHandler) {
				t.Fatalf("expected handler %s, got %s", test.expectedHandler, handler(&opts))
			}
		})
	}
}

func TestHandlerFromStringError(t *testing.T) {
	_, err := handlerFromString("invalid")
	if !errors.Is(err, errInvalidHandler) {
		t.Fatalf("expected error %s, got %s", errInvalidHandler, err)
	}
}

func TestConfigInitialize(t *testing.T) {
	tests := map[string]struct {
		settings    Config
		expectedErr error
	}{
		"debug and text success": {
			settings: Config{
				Level:   "debug",
				Handler: "text",
			},
			expectedErr: nil,
		},
		"default values": {
			settings:    Config{},
			expectedErr: nil,
		},
		"invalid level": {
			settings:    Config{Level: "invalid"},
			expectedErr: errInvalidSettings,
		},
		"invalid handler": {
			settings:    Config{Handler: "invalid"},
			expectedErr: errInvalidSettings,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := test.settings.Initialize()
			if !errors.Is(err, test.expectedErr) {
				t.Fatalf("expected error %s, got %s", test.expectedErr, err)
			}
		})
	}
}
