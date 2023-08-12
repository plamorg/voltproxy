package selection

import (
	"errors"
	"reflect"
	"testing"
)

func TestNewStrategyError(t *testing.T) {
	tests := map[string]struct {
		strategy    string
		max         uint
		expectedErr error
	}{
		"no services": {
			strategy:    "",
			max:         0,
			expectedErr: ErrNoServicesSpecified,
		},
		"invalid strategy": {
			strategy:    "invalid",
			max:         1,
			expectedErr: ErrInvalidStrategy,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewStrategy(test.strategy, test.max)
			if !errors.Is(err, test.expectedErr) {
				t.Fatalf("expected %v, got %v", test.expectedErr, err)
			}
		})
	}
}

func TestNewStrategySuccess(t *testing.T) {
	tests := []struct {
		strategy         string
		expectedStrategy reflect.Type
	}{
		{
			strategy:         "roundRobin",
			expectedStrategy: reflect.TypeOf(&RoundRobin{}),
		},
		{
			strategy:         "random",
			expectedStrategy: reflect.TypeOf(&Random{}),
		},
	}
	for _, test := range tests {
		t.Run(test.strategy, func(t *testing.T) {
			strategy, err := NewStrategy(test.strategy, 5)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if strategy == nil {
				t.Fatalf("expected strategy to be set")
			}
			strategyType := reflect.TypeOf(strategy)
			if strategyType != test.expectedStrategy {
				t.Fatalf("expected strategy to be %v, got %v", test.expectedStrategy, strategyType)
			}
		})
	}
}

func TestRoundRobin(t *testing.T) {
	max := uint(3)
	rr := NewRoundRobin(max)

	expected := []uint{0, 1, 2, 0, 1, 2}
	for i := 0; i < len(expected); i++ {
		next := rr.Select()
		if next != expected[i] {
			t.Fatalf("expected %d, got %d", expected[i], next)
		}
	}
}

func TestRandomSingleService(t *testing.T) {
	trials := 100
	max := uint(1)
	r := NewRandom(max)

	for i := 0; i < trials; i++ {
		next := r.Select()
		if next != 0 {
			t.Fatalf("expected 0, got %d", next)
		}
	}
}
