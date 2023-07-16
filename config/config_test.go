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
			errors.New("service invalid has both container and address")},
		"service with neither address no container": {
			[]byte(`
services:
  wrong:
    `),
			nil,
			errors.New("service wrong has neither container nor address")},
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
