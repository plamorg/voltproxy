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
		"multiple routers": {
			services: serviceConfig{
				"foo": {
					routers: routers{
						Container: &containerInfo{},
						Redirect:  "https://example.com",
					},
				},
			},
			err: errMustHaveOneRouter,
		},
		"invalid strategy": {
			services: serviceConfig{
				"foo": {
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
					routers: routers{
						LoadBalancer: &loadBalancerInfo{
							ServiceNames: []string{"invalid"},
						},
					},
				},
			},
			err: errNoServiceWithName,
		},
		"load balancer tries to load balance itself": {
			services: serviceConfig{
				"foo": {
					routers: routers{
						LoadBalancer: &loadBalancerInfo{
							ServiceNames: []string{"foo"},
						},
					},
				},
			},
			err: errNoServiceWithName,
		},
		"load balancer tries to load balance another load balancer": {
			services: serviceConfig{
				"foo": {
					routers: routers{
						LoadBalancer: &loadBalancerInfo{
							ServiceNames: []string{"bar"},
						},
					},
				},
				"bar": {
					routers: routers{
						LoadBalancer: &loadBalancerInfo{
							ServiceNames: []string{},
						},
					},
				},
			},
			err: errNoServiceWithName,
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
