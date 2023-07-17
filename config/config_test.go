package voltconfig

import (
	"errors"

	"testing"

	"github.com/google/go-cmp/cmp"
)

var equateErrorMessage = cmp.Comparer(func(x, y error) bool {
	if x == nil || y == nil {
		return x == nil && y == nil
	}
	return x.Error() == y.Error()
})

func TestValidateServices(t *testing.T) {
	tests := map[string]struct {
		services map[string]Service
		err      error
	}{
		"no services": {map[string]Service{},
			nil},
		"service with address": {map[string]Service{"a": {Host: "b", Address: "c"}},
			nil},
		"service with container": {map[string]Service{"a": {Host: "b", Container: &container{"c", "d", 0}}},
			nil},
		"service with no container/address": {map[string]Service{"bad": {Host: "b"}},
			errors.New("service \"bad\" must have exactly one of container and address")},
		"service with both container and address": {map[string]Service{"invalid": {Host: "b", Address: "c", Container: &container{"d", "e", 1}}},
			errors.New("service \"invalid\" must have exactly one of container and address")},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if err := validateServices(test.services); !cmp.Equal(test.err, err, equateErrorMessage) {
				t.Errorf("expected error %v got error %v", test.err, err)
			}
		})
	}
}

func TestParse(t *testing.T) {
	tests := map[string]struct {
		data   []byte
		config *Config
		err    error
	}{
		"empty config": {[]byte(``), &Config{}, nil},
		"no services":  {[]byte(`services:`), &Config{}, nil},
		"one service": {
			[]byte(`
services:
  example:
    host: host.example.com
    address: https://example.com
    `),
			&Config{map[string]Service{
				"example": {Host: "host.example.com", Address: "https://example.com"}}},
			nil},
		"two services": {
			[]byte(`
services:
  a:
    host: ahost
    container:
        name: "test"
        network: "net"
        port: 1234
  b:
    host: bhost
    address: https://b.example.com
    `),
			&Config{map[string]Service{
				"a": {Host: "ahost", Container: &container{"test", "net", 1234}},
				"b": {Host: "bhost", Address: "https://b.example.com"},
			}},
			nil},
		"service with both address and container": {
			[]byte(`
services:
  invalid:
    host: invalid.host
    container:
        name: "a"
        network: "b"
        port: 8080
    address: https://invalid.example.com
    `),
			nil,
			errors.New("service \"invalid\" must have exactly one of container and address")},
		"service with neither address no container": {
			[]byte(`
services:
  wrong:
    `),
			nil,
			errors.New("service \"wrong\" must have exactly one of container and address")},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			config, err := Parse(test.data)
			if !cmp.Equal(test.err, err, equateErrorMessage) {
				t.Errorf("expected error %v got error %v", test.err, err)
			}
			if !cmp.Equal(test.config, config) {
				t.Errorf("expected config %v got config %v", test.config, config)
			}
		})
	}

}
