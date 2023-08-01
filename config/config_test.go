package config

import (
	"errors"
	"reflect"

	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/plamorg/voltproxy/dockerapi"
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
		data           []byte
		expectedConfig *Config
		err            error
	}{
		"empty config": {
			[]byte(``),
			&Config{},
			nil,
		},
		"no services": {
			[]byte(`services:`),
			&Config{},
			nil,
		},
		"one service": {
			[]byte(`
services:
  example:
    host: host.example.com
    redirect: https://example.com`),
			&Config{
				serviceMap{
					"example": {
						Host:     "host.example.com",
						TLS:      false,
						Redirect: "https://example.com",
					},
				},
			},
			nil,
		},
		"two services": {
			[]byte(`
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
    redirect: https://b.example.com`),
			&Config{
				serviceMap{
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
			nil,
		},
		"service with both address and container": {
			[]byte(`
services:
  invalid:
    host: invalid.host
    container:
        name: "a"
        network: "b"
        port: 8080
    redirect: https://invalid.example.com`),
			nil,
			errMustHaveOneService,
		},
		"service with neither address no container": {
			[]byte(`
services:
  wrong:
    `),
			nil,
			errMustHaveOneService,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			config, err := Parse(test.data)
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
      ipallow:
        - 127.0.0.1
        - 192.168.1.7
    `)

	expectedConfig := &Config{
		serviceMap{
			"service1": {
				Host:     "service1.example.com",
				Redirect: "https://invalid.example.com",
				Middlewares: &middlewareData{
					IPAllow: middlewares.NewIPAllow([]string{"127.0.0.1", "192.168.1.7"}),
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
				serviceMap{
					"a": {
						Host:     "a",
						Redirect: "b",
					},
				},
			},
			services.List{
				services.NewRedirect("a", nil, "b"),
			},
			nil,
		},
		"with middleware": {
			Config{
				serviceMap{
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
				services.NewRedirect("a", []middlewares.Middleware{middlewares.NewIPAllow(nil)}, "b"),
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

					if service.Host() != expectedService.Host() {
						t.Errorf("expected host %s got host %s", expectedService.Host(), service.Host())
					}
					if remote.String() != expectedRemote.String() {
						t.Errorf("expected remote %s got remote %s", expectedRemote.String(), remote.String())
					}
					if !reflect.DeepEqual(service.Middlewares(), expectedService.Middlewares()) {
						t.Errorf("expected middlewares %v got middlewares %v", expectedService.Middlewares(), service.Middlewares())
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
		serviceMap{
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
		services.NewContainer(adapter, "a", nil, services.ContainerInfo{
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
				serviceMap{
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
				serviceMap{
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
				serviceMap{
					"a": {
						Host:     "a",
						Redirect: "b",
					},
					"c": {
						Host:     "c",
						Redirect: "d",
						TLS:      true,
					},
				},
			},
			[]string{"c"},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			hosts := test.conf.TLSHosts()
			if !reflect.DeepEqual(test.expected, hosts) {
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
		// TODO: add tests for multiple middlewares.
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
