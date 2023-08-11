package middlewares

import (
	"reflect"
	"testing"
)

func TestConfigList(t *testing.T) {
	tests := map[string]struct {
		config   Middlewares
		expected []Middleware
	}{
		"no middlewares": {
			Middlewares{},
			nil,
		},
		"one middleware": {
			Middlewares{
				IPAllow: NewIPAllow([]string{"a"}),
			},
			[]Middleware{
				NewIPAllow([]string{"a"}),
			},
		},
		"multiple middlewares": {
			Middlewares{
				IPAllow: NewIPAllow([]string{"a"}),
				AuthForward: &AuthForward{
					Address:         "auth server",
					RequestHeaders:  []string{"1", "2"},
					ResponseHeaders: []string{"3"},
				},
			},
			[]Middleware{
				NewIPAllow([]string{"a"}),
				&AuthForward{
					Address:         "auth server",
					RequestHeaders:  []string{"1", "2"},
					ResponseHeaders: []string{"3"},
				},
			},
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			middlewares := test.config.List()
			if !reflect.DeepEqual(test.expected, middlewares) {
				t.Errorf("expected %v got %v", test.expected, middlewares)
			}
		})
	}
}
