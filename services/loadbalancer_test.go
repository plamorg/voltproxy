package services

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/plamorg/voltproxy/services/health"
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
			expectedErr: errNoServicesSpecified,
		},
		"invalid strategy": {
			services: []Service{nil},
			info: LoadBalancerInfo{
				Strategy:     "invalid",
				ServiceNames: []string{"foo"},
			},
			expectedErr: errInvalidStrategy,
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

type mockService struct {
	data Data
}

func newMockService(host string, up bool) *mockService {
	return &mockService{
		data: Data{
			Host:   host,
			Health: health.Always(up),
		},
	}
}

func (m *mockService) Data() *Data {
	return &m.data
}

func (m *mockService) Remote(http.ResponseWriter, *http.Request) (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://%s", m.data.Host))
}

func TestLoadBalancerNextService(t *testing.T) {
	tests := map[string]struct {
		services     []Service
		expectedHost string
	}{
		"one service": {
			services:     []Service{newMockService("foo", true)},
			expectedHost: "foo",
		},
		"one failed service": {
			services:     []Service{newMockService("foo", false)},
			expectedHost: "foo",
		},
		"two services": {
			services: []Service{
				newMockService("foo", true),
				newMockService("bar", true),
			},
			expectedHost: "foo",
		},
		"skip failed services": {
			services: []Service{
				newMockService("foo", false),
				newMockService("bar", false),
				newMockService("baz", true),
				newMockService("baz2", true),
				newMockService("baz3", false),
			},
			expectedHost: "baz",
		},
		"all failed services": {
			services: []Service{
				newMockService("foo", false),
				newMockService("bar", false),
				newMockService("baz", false),
			},
			expectedHost: "baz",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lb := LoadBalancer{
				services: test.services,
				strategy: selection.NewRoundRobin(uint(len(test.services))),
			}
			next := lb.nextService(nil, nil)
			host := test.services[next].Data().Host
			if host != test.expectedHost {
				t.Fatalf("expected %s, got %s", test.expectedHost, host)
			}
		})
	}
}

func TestLoadBalancerFailover(t *testing.T) {
	allUp := []Service{
		newMockService("foo", true),
		newMockService("bar", true),
		newMockService("baz", true),
	}

	fooDown := []Service{
		newMockService("foo", false),
		newMockService("bar", true),
		newMockService("baz", true),
	}

	barDown := []Service{
		newMockService("foo", true),
		newMockService("bar", false),
		newMockService("baz", true),
	}

	lb := LoadBalancer{
		services: allUp,
		strategy: selection.NewFailover(uint(len(allUp))),
	}

	tests := []struct {
		name     string
		services []Service
		expected string
	}{
		{
			name:     "All services up",
			services: allUp,
			expected: "foo",
		},
		{
			name:     "Foo service down",
			services: fooDown,
			expected: "bar",
		},
		{
			name:     "All services up again",
			services: allUp,
			expected: "foo",
		},
		{
			name:     "Bar service down",
			services: barDown,
			expected: "foo",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lb.services = test.services
			next, err := lb.Remote(nil, nil)
			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			host := next.Host
			if host != test.expected {
				t.Fatalf("expected %s, got %s", test.expected, host)
			}
		})
	}
}
