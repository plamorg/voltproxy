package config

import (
	"errors"

	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestValidateServices(t *testing.T) {
	tests := map[string]struct {
		services serviceList
		err      error
	}{
		"no services": {
			serviceList{},
			nil,
		},
		"service with address": {
			serviceList{"a": {Host: "b", Redirect: "c"}},
			nil,
		},
		"service with container": {
			serviceList{"a": {Host: "b", Container: &containerInfo{"c", "d", 0}}},
			nil,
		},
		"service with TLS": {
			serviceList{"bad": {Host: "a", Redirect: "b", TLS: true}},
			nil,
		},
		"service with no container/address": {
			serviceList{"bad": {Host: "b"}},
			errMustHaveOneService,
		},
		"service with both container and address": {
			serviceList{"invalid": {Host: "b", Redirect: "c", Container: &containerInfo{"d", "e", 1}}},
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
				serviceList{
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
				serviceList{
					"a": {
						Host:      "ahost",
						TLS:       false,
						Container: &containerInfo{"test", "net", 1234},
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
