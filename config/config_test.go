package config

import (
	"errors"
	"reflect"
	"slices"

	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/plamorg/voltproxy/dockerapi"
	"github.com/plamorg/voltproxy/logging"
	"github.com/plamorg/voltproxy/middlewares"
	"github.com/plamorg/voltproxy/services"
)

func TestValidateServices(t *testing.T) {
	tests := map[string]struct {
		services serviceMap
		err      error
	}{
		"no services": {
			serviceMap{},
			nil,
		},
		"service with address": {
			serviceMap{"a": {Host: "b", Redirect: "c"}},
			nil,
		},
		"service with container": {
			serviceMap{
				"a": {
					Host:      "b",
					Container: &services.ContainerInfo{Name: "c", Network: "d", Port: 0},
				},
			},
			nil,
		},
		"service with TLS": {
			serviceMap{"secure": {Host: "a", Redirect: "b", TLS: true}},
			nil,
		},
		"service with middleware": {
			serviceMap{
				"mid": {
					Host:     "host",
					Redirect: "https://example.com",
					Middlewares: &middlewareData{
						IPAllow: middlewares.NewIPAllow([]string{"172.20.0.1"}),
					}}},
			nil,
		},
		"service with no container/address": {
			serviceMap{"bad": {Host: "b"}},
			errMustHaveOneService,
		},
		"service with both container and address": {
			serviceMap{
				"invalid": {Host: "b",
					Redirect: "c",
					Container: &services.ContainerInfo{
						Name:    "d",
						Network: "e",
						Port:    1,
					},
				},
			},
			errMustHaveOneService,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateServices(test.services); !errors.Is(err, test.err) {
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
						Host:     "host.example.com",
						TLS:      false,
						Redirect: "https://example.com",
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
						Host:      "ahost",
						TLS:       false,
						Container: &services.ContainerInfo{Name: "test", Network: "net", Port: 1234},
					},
					"b": {
						Host:     "bhost",
						TLS:      true,
						Redirect: "https://b.example.com",
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
		"": {
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
				Host:     "service1.example.com",
				Redirect: "https://invalid.example.com",
				Middlewares: &middlewareData{
					IPAllow: middlewares.NewIPAllow([]string{"127.0.0.1", "192.168.1.7"}),
					AuthForward: &middlewares.AuthForward{
						Address:         "https://auth.example.com",
						XForwarded:      true,
						RequestHeaders:  []string{},
						ResponseHeaders: []string{"X-Auth-Response-Header"},
					},
				},
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
						Host:     "a",
						Redirect: "b",
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
						Host:     "a",
						Redirect: "b",
						Middlewares: &middlewareData{
							IPAllow: middlewares.NewIPAllow(nil),
						},
					},
				},
			},
			services.List{
				services.NewRedirect(services.Config{
					Host:        "a",
					Middlewares: []middlewares.Middleware{middlewares.NewIPAllow(nil)},
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

			if len(services) == len(test.expected) {
				for i, service := range services {
					remote, err := service.Remote()
					expectedService := test.expected[i]
					expectedRemote, expectedErr := expectedService.Remote()

					if !reflect.DeepEqual(service.Config(), expectedService.Config()) {
						t.Errorf("expected service %v got service %v", expectedService.Config(), service.Config())
					}
					if remote.String() != expectedRemote.String() {
						t.Errorf("expected remote %s got remote %s", expectedRemote.String(), remote.String())
					}
					if err != expectedErr {
						t.Errorf("expected error %v got error %v", expectedErr, err)
					}
				}
			} else {
				t.Fatalf("expected %d services got %d", len(test.expected), len(services))
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
				Host: "a",
				Container: &services.ContainerInfo{
					Name:    "b",
					Network: "c",
					Port:    1234,
				},
			},
		},
	}

	expectedServices := services.List{
		services.NewContainer(adapter, services.Config{Host: "a"}, services.ContainerInfo{
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
						Host:     "a",
						Redirect: "b",
					},
				},
			},
			nil,
		},
		"one redirect with TLS": {
			Config{
				Services: serviceMap{
					"a": {
						Host:     "a",
						Redirect: "b",
						TLS:      true,
					},
				},
			},
			[]string{"a"},
		},
		"multiple redirects": {
			Config{
				Services: serviceMap{
					"a": {
						Host:     "a",
						Redirect: "b",
					},
					"b": {
						Host:     "b",
						Redirect: "c",
						TLS:      true,
					},
					"c": {
						Host:     "c",
						Redirect: "d",
						TLS:      true,
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

func TestMiddlewareDataList(t *testing.T) {
	tests := map[string]struct {
		data     middlewareData
		expected []middlewares.Middleware
	}{
		"no middlewares": {
			middlewareData{},
			nil,
		},
		"one middleware": {
			middlewareData{
				IPAllow: middlewares.NewIPAllow([]string{"a"}),
			},
			[]middlewares.Middleware{
				middlewares.NewIPAllow([]string{"a"}),
			},
		},
		"multiple middlewares": {
			middlewareData{
				IPAllow: middlewares.NewIPAllow([]string{"a"}),
				AuthForward: &middlewares.AuthForward{
					Address:         "auth server",
					RequestHeaders:  []string{"1", "2"},
					ResponseHeaders: []string{"3"},
				},
			},
			[]middlewares.Middleware{
				middlewares.NewIPAllow([]string{"a"}),
				&middlewares.AuthForward{
					Address:         "auth server",
					RequestHeaders:  []string{"1", "2"},
					ResponseHeaders: []string{"3"},
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			middlewares := test.data.List()
			if !reflect.DeepEqual(test.expected, middlewares) {
				t.Errorf("expected middlewares %v got middlewares %v", test.expected, middlewares)
			}
		})
	}
}
