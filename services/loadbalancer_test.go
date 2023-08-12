package services

import (
	"errors"
	"testing"

	"github.com/plamorg/voltproxy/services/selection"
)

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

	if _, ok := lb.strategy.(*selection.RoundRobin); !ok {
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
			expectedErr: selection.ErrNoServicesSpecified,
		},
		"invalid strategy": {
			services: []Service{nil},
			info: LoadBalancerInfo{
				Strategy:     "invalid",
				ServiceNames: []string{"foo"},
			},
			expectedErr: selection.ErrInvalidStrategy,
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
