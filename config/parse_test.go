package config

import (
	"errors"
	"testing"
)

func TestUniqueHosts(t *testing.T) {
	tests := map[string]struct {
		serviceConfig serviceConfig
		want          bool
	}{
		"empty": {
			serviceConfig: serviceConfig{},
			want:          true,
		},
		"unique": {
			serviceConfig: serviceConfig{
				"foo": {Host: "example.com"},
				"bar": {Host: "foo.example.com"},
				"baz": {Host: "baz.example.com"},
			},
			want: true,
		},
		"duplicate": {
			serviceConfig: serviceConfig{
				"foo": {Host: "example.com"},
				"bar": {Host: "example.com"},
			},
			want: false,
		},
		"ignore empty host": {
			serviceConfig: serviceConfig{
				"foo": {Host: ""},
				"bar": {Host: ""},
			},
			want: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := uniqueHosts(test.serviceConfig)
			if got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestConfigServicesError(t *testing.T) {
	tests := map[string]struct {
		services serviceConfig
		err      error
	}{
		"duplicate host": {
			services: serviceConfig{
				"foo": {Host: "example.com"},
				"bar": {Host: "example.com"},
			},
			err: errDuplicateHost,
		},
		"multiple services": {
			services: serviceConfig{
				"foo": {
					Host: "example.com",
					routers: routers{
						Container: &containerInfo{},
						Redirect:  "https://example.com",
					},
				},
			},
			err: errMustHaveOneService,
		},
		"invalid strategy": {
			services: serviceConfig{
				"foo": {
					Host: "example.com",
					routers: routers{
						LoadBalancer: &loadBalancerInfo{
							Strategy: "invalid",
						},
					},
				},
			},
			err: errInvalidConfig,
		},
		"invalid redirect": {
			services: serviceConfig{
				"foo": {
					Host: "example.com",
					routers: routers{
						Redirect: "$%^&*",
					},
				},
			},
			err: errInvalidConfig,
		},
		"load balancer with invalid service name": {
			services: serviceConfig{
				"foo": {
					Host: "example.com",
					routers: routers{
						LoadBalancer: &loadBalancerInfo{
							ServiceNames: []string{"invalid"},
						},
					},
				},
			},
			err: errInvalidConfig,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			conf := Config{
				ServiceConfig: test.services,
			}
			_, err := conf.Services(nil)
			if !errors.Is(err, test.err) {
				t.Errorf("got %v, want %v", err, test.err)
			}
		})
	}
}
