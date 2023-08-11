package config

import (
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/plamorg/voltproxy/dockerapi"
	"github.com/plamorg/voltproxy/logging"
	"github.com/plamorg/voltproxy/middlewares"
	"github.com/plamorg/voltproxy/services"
)

func TestServiceMapValidate(t *testing.T) {
	tests := map[string]struct {
		services serviceMap
		err      error
	}{
		"no services": {
			serviceMap{},
			nil,
		},
		"service with address": {
			serviceMap{
				"a": {
					Config:   services.Config{Host: "b"},
					Services: services.Services{Redirect: "c"},
				},
			},
			nil,
		},
		"service with container": {
			serviceMap{
				"a": {
					Config:   services.Config{Host: "b"},
					Services: services.Services{Container: &services.ContainerInfo{Name: "c", Network: "d", Port: 0}},
				},
			},
			nil,
		},
		"service with TLS": {
			serviceMap{
				"secure": {
					Config:   services.Config{Host: "a", TLS: true},
					Services: services.Services{Redirect: "b"},
				},
			},
			nil,
		},
		"service with middleware": {
			serviceMap{
				"mid": {
					Config: services.Config{
						Host: "host",
						Middlewares: &middlewares.Middlewares{
							IPAllow: middlewares.NewIPAllow([]string{"172.20.0.1"}),
						},
					},
					Services: services.Services{Redirect: "https://example.com"},
				},
			},
			nil,
		},
		"service with no container/address": {
			serviceMap{"bad": {Config: services.Config{Host: "b"}}},
			errMustHaveOneService,
		},
		"service with both container and address": {
			serviceMap{
				"invalid": {
					Config: services.Config{Host: "b"},
					Services: services.Services{
						Redirect:  "c",
						Container: &services.ContainerInfo{Name: "d", Network: "e", Port: 1},
					},
				},
			},
			errMustHaveOneService,
		},
		"services with duplicate host": {
			serviceMap{
				"a": {
					Config:   services.Config{Host: "b"},
					Services: services.Services{Redirect: "c"},
				},
				"d": {
					Config:   services.Config{Host: "b"},
					Services: services.Services{Redirect: "c"},
				},
			},
			errDuplicateHost,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := test.services.validate(); !errors.Is(err, test.err) {
				t.Errorf("expected error %v got error %v", test.err, err)
			}
		})
	}
}

func TestParseInvalidSyntax(t *testing.T) {
	data := []byte(`
services:
    - 123
    - "abc"`)
	_, err := Parse(data)

	if err == nil {
		t.Errorf("expected error, got nil")
	}
}

func TestParse(t *testing.T) {
	tests := map[string]struct {
		config         string
		expectedConfig *Config
		err            error
	}{
		"empty config": {
			config:         "",
			expectedConfig: nil,
			err:            errInvalidConfig,
		},
		"no services": {
			config:         "services:",
			expectedConfig: &Config{},
			err:            nil,
		},
		"one service": {
			config: `
services:
  example:
    host: host.example.com
    redirect: https://example.com`,
			expectedConfig: &Config{
				Services: serviceMap{
					"example": {
						Config: services.Config{
							Host: "host.example.com",
							TLS:  false,
						},
						Services: services.Services{Redirect: "https://example.com"},
					},
				},
			},
			err: nil,
		},
		"two services": {
			config: `
services:
  a:
    host: ahost
    tls: false
    container:
        name: "test"
        network: "net"
        port: 1234
  b:
    host: bhost
    tls: true
    redirect: https://b.example.com`,
			expectedConfig: &Config{
				Services: serviceMap{
					"a": {
						Config: services.Config{
							Host: "ahost",
							TLS:  false,
						},
						Services: services.Services{
							Container: &services.ContainerInfo{Name: "test", Network: "net", Port: 1234},
						},
					},
					"b": {
						Config: services.Config{
							Host: "bhost",
							TLS:  true,
						},
						Services: services.Services{
							Redirect: "https://b.example.com",
						},
					},
				},
			},
			err: nil,
		},
		"service with both address and container": {
			config: `
services:
  invalid:
    host: invalid.host
    container:
        name: "a"
        network: "b"
        port: 8080
    redirect: https://invalid.example.com`,
			expectedConfig: nil,
			err:            errMustHaveOneService,
		},
		"service with neither address no container": {
			config: `
services:
  wrong:
    `,
			expectedConfig: nil,
			err:            errMustHaveOneService,
		},
		"service with invalid middleware": {
			config: `
services:
  service1:
    host: service1.example.com
    redirect: https://invalid.example.com
    middlewares:
      thisMiddlewareDoesNotExist:
        - "test"
        `,
			expectedConfig: nil,
			err:            errInvalidConfig,
		},
		"log configuration": {
			config: `
log:
  level: "warn"
  handler: "json"

services:
        `,
			expectedConfig: &Config{
				Log: logging.Config{
					Level:   "warn",
					Handler: "json",
				},
			},
			err: nil,
		},
		"read timeout": {
			config: `readTimeout: 10s`,
			expectedConfig: &Config{
				ReadTimeout: 10 * time.Second,
			},
			err: nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			config, err := Parse([]byte(test.config))
			if !errors.Is(err, test.err) {
				t.Fatalf("expected error %v got error %v", test.err, err)
			}
			if !reflect.DeepEqual(test.expectedConfig, config) {
				t.Errorf("expected config %v got config %v", test.expectedConfig, config)
			}
		})
	}
}

func TestParseWithMiddlewares(t *testing.T) {
	data := []byte(`
services:
  service1:
    host: service1.example.com
    redirect: https://invalid.example.com
    middlewares:
      ipAllow:
        - 127.0.0.1
        - 192.168.1.7
      authForward:
        address: https://auth.example.com
        xForwarded: true
        requestHeaders: []
        responseHeaders: ["X-Auth-Response-Header"]
    `)

	expectedConfig := &Config{
		Services: serviceMap{
			"service1": {
				Config: services.Config{
					Host: "service1.example.com",
					Middlewares: &middlewares.Middlewares{
						IPAllow: middlewares.NewIPAllow([]string{"127.0.0.1", "192.168.1.7"}),
						AuthForward: &middlewares.AuthForward{
							Address:         "https://auth.example.com",
							XForwarded:      true,
							RequestHeaders:  []string{},
							ResponseHeaders: []string{"X-Auth-Response-Header"},
						},
					},
				},
				Services: services.Services{Redirect: "https://invalid.example.com"},
			},
		},
	}

	config, err := Parse(data)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if !reflect.DeepEqual(expectedConfig, config) {
		t.Errorf("expected config %v got config %v", expectedConfig, config)
	}
}

func TestConfigServiceList(t *testing.T) {
	tests := map[string]struct {
		conf     Config
		expected services.List
		err      error
	}{
		"no services": {
			Config{},
			nil,
			nil,
		},
		"one redirect": {
			Config{
				Services: serviceMap{
					"a": {
						Config: services.Config{
							Host: "a",
						},
						Services: services.Services{Redirect: "b"},
					},
				},
			},
			services.List{
				services.NewRedirect(services.Config{Host: "a"}, "b"),
			},
			nil,
		},
		"with middleware": {
			Config{
				Services: serviceMap{
					"a": {
						Config: services.Config{
							Host: "a",
							Middlewares: &middlewares.Middlewares{
								IPAllow: middlewares.NewIPAllow(nil),
							},
						},
						Services: services.Services{Redirect: "b"},
					},
				},
			},
			services.List{
				services.NewRedirect(services.Config{
					Host: "a",
					Middlewares: &middlewares.Middlewares{
						IPAllow: middlewares.NewIPAllow(nil),
					},
				}, "b"),
			},
			nil,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			services, err := test.conf.ServiceList(dockerapi.NewMock(nil))
			if !errors.Is(err, test.err) {
				t.Errorf("expected error %v got error %v", test.err, err)
			}

			if !reflect.DeepEqual(test.expected, services) {
				t.Errorf("expected services %v got services %v", test.expected, services)
			}
		})
	}
}

func TestConfigServiceListWithContainers(t *testing.T) {
	containers := []types.Container{
		{
			Names: []string{"b"},
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"c": {
						IPAddress: "1234",
					},
				},
			},
		},
	}
	adapter := dockerapi.NewMock(containers)

	conf := Config{
		Services: serviceMap{
			"a": {
				Config: services.Config{Host: "a"},
				Services: services.Services{
					Container: &services.ContainerInfo{
						Name:    "b",
						Network: "c",
						Port:    1234,
					},
				},
			},
		},
	}

	expectedServices := services.List{
		services.NewContainer(services.Config{Host: "a"}, adapter, services.ContainerInfo{
			Name:    "b",
			Network: "c",
			Port:    1234,
		}),
	}

	services, err := conf.ServiceList(adapter)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}

	if !reflect.DeepEqual(expectedServices, services) {
		t.Errorf("expected services %v got services %v", expectedServices, services)
	}
}

func TestConfigServiceListLoadBalancerError(t *testing.T) {
	tests := map[string]struct {
		lbInfo services.LoadBalancerInfo
	}{
		"no service with name": {
			lbInfo: services.LoadBalancerInfo{
				ServiceNames: []string{"foo"},
			},
		},
		"empty service names": {
			lbInfo: services.LoadBalancerInfo{
				ServiceNames: nil,
			},
		},
		"invalid strategy": {
			lbInfo: services.LoadBalancerInfo{
				ServiceNames: nil,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			conf := Config{
				Services: serviceMap{
					"lb": {
						Config: services.Config{Host: "a"},
						Services: services.Services{
							LoadBalancer: &test.lbInfo,
						},
					},
				},
			}

			_, err := conf.ServiceList(dockerapi.NewMock(nil))

			if !errors.Is(err, errInvalidConfig) {
				t.Errorf("expected error %v got error %v", errInvalidConfig, err)
			}
		})
	}
}

func TestConfigTLSHosts(t *testing.T) {
	tests := map[string]struct {
		conf     Config
		expected []string
	}{
		"no services": {
			Config{},
			nil,
		},
		"one redirect": {
			Config{
				Services: serviceMap{
					"a": {
						Config: services.Config{Host: "a"},
					},
				},
			},
			nil,
		},
		"one redirect with TLS": {
			Config{
				Services: serviceMap{
					"a": {
						Config: services.Config{
							Host: "a",
							TLS:  true,
						},
					},
				},
			},
			[]string{"a"},
		},
		"multiple redirects": {
			Config{
				Services: serviceMap{
					"a": {
						Config: services.Config{Host: "a"},
					},
					"b": {
						Config: services.Config{
							Host: "b",
							TLS:  true,
						},
					},
					"c": {
						Config: services.Config{
							Host: "c",
							TLS:  true,
						},
					},
				},
			},
			[]string{"b", "c"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			hosts := test.conf.TLSHosts()
			slices.Sort(hosts)
			slices.Sort(test.expected)
			if !slices.Equal(hosts, test.expected) {
				t.Errorf("expected hosts %v got hosts %v", test.expected, hosts)
			}
		})
	}
}
