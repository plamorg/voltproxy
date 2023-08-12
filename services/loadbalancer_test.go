package services

import (
	"errors"
	"testing"
)

func TestRoundRobin(t *testing.T) {
	items := []string{"a", "b", "c"}
	rr := newRoundRobin(items)
	for _, item := range items {
		if rr.next() != item {
			t.Fatalf("expected %s, got %s", item, rr.next())
		}
	}

	// Check that it loops around.
	if rr.next() != "a" {
		t.Fatalf("expected %s, got %s", "a", rr.next())
	}
}

func TestNewLoadBalancerDefaultRoundRobin(t *testing.T) {
	services := []Service{nil}
	info := LoadBalancerInfo{
		ServiceNames: []string{"foo"},
	}

	lb, err := NewLoadBalancer(Data{}, services, info)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if lb.strategy == nil {
		t.Fatalf("expected strategy to be set")
	}

	if _, ok := lb.strategy.(*roundRobin[Service]); !ok {
		t.Fatalf("expected strategy to be a round robin")
	}
}

func TestNewLoadBalancerError(t *testing.T) {
	tests := map[string]struct {
		services    []Service
		info        LoadBalancerInfo
		expectedErr error
	}{
		"no services": {
			services:    []Service{},
			info:        LoadBalancerInfo{},
			expectedErr: errNoServicesSpecified,
		},
		"invalid strategy": {
			services: []Service{nil},
			info: LoadBalancerInfo{
				Strategy:     "invalid",
				ServiceNames: []string{"foo"},
			},
			expectedErr: errInvalidLoadBalancerStrategy,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := NewLoadBalancer(Data{}, test.services, test.info)
			if !errors.Is(err, test.expectedErr) {
				t.Fatalf("expected error %v, got %v", test.expectedErr, err)
			}
		})
	}
}
