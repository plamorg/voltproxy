package config

import (
	"errors"
	"slices"
	"testing"
)

func TestEnsureOneRouter(t *testing.T) {
	tests := map[string]struct {
		routers routers
		want    error
	}{
		"empty": {
			routers: routers{},
			want:    errMustHaveOneRouter,
		},
		"container": {
			routers: routers{
				Container: &containerInfo{},
			},
			want: nil,
		},
		"redirect": {
			routers: routers{
				Redirect: "https://example.com",
			},
			want: nil,
		},
		"load balancer": {
			routers: routers{
				LoadBalancer: &loadBalancerInfo{},
			},
			want: nil,
		},
		"multiple": {
			routers: routers{
				Container: &containerInfo{},
				Redirect:  "https://example.com",
			},
			want: errMustHaveOneRouter,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := test.routers.ensureOneRouter()
			if !errors.Is(err, test.want) {
				t.Errorf("got %v, want %v", err, test.want)
			}
		})
	}
}

func TestNewEmptyConfig(t *testing.T) {
	_, err := New([]byte(``))
	if !errors.Is(err, errInvalidConfig) {
		t.Errorf("got %v, want %v", err, errInvalidConfig)
	}
}

func TestConfigTLSHosts(t *testing.T) {
	tests := map[string]struct {
		services serviceConfig
		want     []string
	}{
		"empty": {
			services: serviceConfig{},
			want:     nil,
		},
		"no host": {
			services: serviceConfig{
				"foo": {},
				"bar": {},
			},
			want: nil,
		},
		"ignore non-TLS": {
			services: serviceConfig{
				"foo": {Host: "example.com"},
				"bar": {},
			},
			want: []string{},
		},
		"with TLS": {
			services: serviceConfig{
				"foo": {TLS: true, Host: "example.com"},
				"bar": {TLS: true, Host: "foo.example.com"},
				"baz": {TLS: true, Host: "baz.example.com"},
			},
			want: []string{
				"example.com",
				"foo.example.com",
				"baz.example.com",
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			config := &Config{
				ServiceConfig: test.services,
			}
			got := config.TLSHosts()

			slices.Sort(got)
			slices.Sort(test.want)
			if !slices.Equal(got, test.want) {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}
