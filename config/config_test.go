package config

import (
	"errors"
	"reflect"

	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/google/go-cmp/cmp"
	"github.com/plamorg/voltproxy/dockerapi"
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
			serviceMap{"bad": {Host: "a", Redirect: "b", TLS: true}},
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
				t.Errorf("expected error %v got error %v", test.err, err)
			}
			if !cmp.Equal(test.expectedConfig, config) {
				t.Errorf("expected config %v got config %v", test.expectedConfig, config)
			}
		})
	}
}

func TestServiceList(t *testing.T) {
	tests := map[string]struct {
		conf     Config
		expected []services.Service
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
			[]services.Service{
				services.NewRedirect("a", "b"),
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

					var (
						hostEquals   = service.Host() == expectedService.Host()
						remoteEquals = remote.String() == expectedRemote.String()
						errEquals    = err == expectedErr
					)

					if !hostEquals || !remoteEquals || !errEquals {
						t.Errorf("expected services %v got services %v", test.expected, services)
					}
				}
			} else {
				t.Fatalf("expected %d services got %d", len(test.expected), len(services))
			}
		})
	}

}

func TestServiceListWithContainers(t *testing.T) {
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

	expectedServices := []services.Service{
		services.NewContainer(adapter, "a", services.ContainerInfo{
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

func TestTLSHosts(t *testing.T) {
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
