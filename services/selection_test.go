package services

import (
	"errors"
	"fmt"
	"math/rand"
	"reflect"
	"slices"
	"testing"

	"github.com/plamorg/voltproxy/services/health"
)

func TestNewStrategy(t *testing.T) {
	tests := []struct {
		strategy string
		expected Strategy
	}{
		{"", &RoundRobin{next: 0}},
		{"failover", &Failover{}},
		{"roundRobin", &RoundRobin{next: 0}},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("strategy \"%s\"", test.strategy), func(t *testing.T) {
			actual, err := NewStrategy(test.strategy)
			if err != nil {
				t.Errorf("expected nil, got %v", err)
			}
			if !reflect.DeepEqual(actual, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}

	t.Run("random", func(t *testing.T) {
		actual, err := NewStrategy("random")
		if err != nil {
			t.Errorf("expected nil, got %v", err)
		}
		if random, ok := actual.(*Random); !ok || random.rng == nil {
			t.Errorf("expected Random, got %v", actual)
		}
	})
}

func TestNewStrategyError(t *testing.T) {
	_, err := NewStrategy("invalid")
	if !errors.Is(err, errInvalidStrategy) {
		t.Errorf("expected %v, got %v", errInvalidStrategy, err)
	}
}

func TestFailoverSelect(t *testing.T) {
	tests := map[string]struct {
		services []*Service
		expected []int
	}{
		"no services": {
			services: []*Service{},
			expected: []int{0, 0, 0},
		},
		"first service": {
			services: []*Service{
				{Health: health.Always(true)},
				{Health: health.Always(true)},
				{Health: health.Always(true)},
			},
			expected: []int{0, 0, 0},
		},
		"skip failing services": {
			services: []*Service{
				{Health: health.Always(false)},
				{Health: health.Always(false)},
				{Health: health.Always(true)},
			},
			expected: []int{2, 2, 2},
		},
		"all failing services": {
			services: []*Service{
				{Health: health.Always(false)},
				{Health: health.Always(false)},
			},
			expected: []int{0, 0, 0},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := make([]int, 0, len(test.expected))
			f := &Failover{}
			for i := 0; i < len(test.expected); i++ {
				actual = append(actual, f.Select(test.services, nil))
			}
			if !slices.Equal(actual, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestRoundRobinSelect(t *testing.T) {
	tests := map[string]struct {
		services []*Service
		expected []int
	}{
		"no services": {
			services: []*Service{},
			expected: []int{0, 0, 0},
		},
		"wrap around": {
			services: []*Service{
				{Health: health.Always(true)},
				{Health: health.Always(true)},
				{Health: health.Always(true)},
			},
			expected: []int{0, 1, 2, 0, 1, 2, 0, 1, 2},
		},
		"varying health": {
			services: []*Service{
				{Health: health.Always(false)},
				{Health: health.Always(true)},
				{Health: health.Always(true)},
				{Health: health.Always(false)},
				{Health: health.Always(false)},
				{Health: health.Always(true)},
				{Health: health.Always(false)},
			},
			expected: []int{1, 2, 5, 1, 2, 5, 1, 2, 5},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := make([]int, 0, len(test.expected))
			r := &RoundRobin{next: 0}
			for i := 0; i < len(test.expected); i++ {
				actual = append(actual, r.Select(test.services, nil))
			}
			if !slices.Equal(actual, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}

func TestRandomSelect(t *testing.T) {
	tests := map[string]struct {
		services []*Service
		expected []int
	}{
		"no services": {
			services: []*Service{},
			expected: []int{0, 0, 0},
		},
		"all down": {
			services: []*Service{
				{Health: health.Always(false)},
				{Health: health.Always(false)},
				{Health: health.Always(false)},
			},
			expected: []int{0, 0, 0},
		},
		"varying health": {
			services: []*Service{
				{Health: health.Always(false)},
				{Health: health.Always(true)}, // 1
				{Health: health.Always(true)}, // 2
				{Health: health.Always(false)},
				{Health: health.Always(false)},
				{Health: health.Always(true)}, // 5
				{Health: health.Always(false)},
				{Health: health.Always(true)}, // 7
			},
			expected: []int{5, 5, 2, 5, 7, 1, 7, 2, 1},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := make([]int, 0, len(test.expected))
			r := &Random{rng: rand.New(rand.NewSource(0)).Intn} // #nosec
			for i := 0; i < len(test.expected); i++ {
				actual = append(actual, r.Select(test.services, nil))
			}
			if !slices.Equal(actual, test.expected) {
				t.Errorf("expected %v, got %v", test.expected, actual)
			}
		})
	}
}
