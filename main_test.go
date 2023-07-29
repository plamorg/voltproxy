package main

import (
	"errors"
	"testing"

	"github.com/plamorg/voltproxy/services"
)

func TestFindServiceWithHostSuccess(t *testing.T) {
	tests := map[string]struct {
		host           string
		services       []services.Service
		expectedHost   string
		expectedRemote string
	}{
		"single service": {
			"example.com",
			[]services.Service{services.NewRedirect("example.com", nil, "remoteA")},
			"example.com", "remoteA",
		},
		"multiple services": {
			"this.example.com",
			[]services.Service{
				services.NewRedirect("example.com", nil, "remoteA"),
				services.NewRedirect("this.example.com", nil, "remoteC"),
				services.NewRedirect("another.example.com", nil, "remoteB"),
			},
			"this.example.com", "remoteC",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			service, err := findServiceWithHost(test.host, test.services)
			if err != nil {
				t.Error(err)
			}

			remote, _ := service.Remote()

			if service.Host() != test.expectedHost || remote.String() != test.expectedRemote {
				t.Errorf("got %s, %s; expected %s, %s", service.Host(), remote.String(), test.expectedHost, test.expectedRemote)
			}
		})
	}

}

func TestFindServiceWithHostFails(t *testing.T) {
	tests := map[string]struct {
		host     string
		services []services.Service
		err      error
	}{
		"empty services raises error": {
			"example.com",
			[]services.Service{},
			errNoServiceWithHost,
		},
		"service not found": {
			"example.com",
			[]services.Service{
				services.NewRedirect("another.example.com", nil, "remoteA"),
				services.NewRedirect("com", nil, "remoteB"),
			},
			errNoServiceWithHost,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := findServiceWithHost(test.host, test.services)
			if !errors.Is(err, test.err) {
				t.Errorf("got %v, expected %v", err, test.err)
			}
		})
	}
}

func TestContains(t *testing.T) {
	tests := map[string]struct {
		slice []string
		item  string
		found bool
	}{
		"empty slice": {
			[]string{},
			"example.com",
			false,
		},
		"item not found": {
			[]string{"example.com", "another.example.com"},
			"this.example.com",
			false,
		},
		"item found": {
			[]string{"example.com", "another.example.com"},
			"example.com",
			true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			found := contains(test.slice, test.item)
			if found != test.found {
				t.Errorf("got %t, expected %t", found, test.found)
			}
		})
	}

}
