package services

import (
	"errors"
	"testing"
)

func TestRoundRobin(t *testing.T) {
	max := uint(3)
	rr := newRoundRobin(max)

	expected := []uint{0, 1, 2, 0, 1, 2}
	for i := 0; i < len(expected); i++ {
		next := rr.next()
		if next != expected[i] {
			t.Fatalf("expected %d, got %d", expected[i], next)
		}
	}
}

func TestGenerateCookieName(t *testing.T) {
	host := "foo.example.com"
	expected := "fb7746954d615d23"
	cookieName := generateCookieName(host)
	if cookieName != expected {
		t.Fatalf("expected %s, got %s", expected, cookieName)
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

	if _, ok := lb.strategy.(*roundRobin); !ok {
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
