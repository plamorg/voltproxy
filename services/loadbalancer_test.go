package services

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/plamorg/voltproxy/services/health"
)

func TestGenerateCookieName(t *testing.T) {
	host := "foo.example.com"
	expected := "fb7746954d615d23"
	cookieName := generateCookieName(host)
	if cookieName != expected {
		t.Fatalf("expected %s, got %s", expected, cookieName)
	}
}

func TestLoadBalancerPersistentService(t *testing.T) {
	lb := NewLoadBalancer("host", &Failover{}, true, []Service{})
	tests := map[string]struct {
		cookie      *http.Cookie
		services    []Service
		expectedURL string
		expectedErr error
	}{
		"no services": {
			cookie: &http.Cookie{
				Name:  lb.cookieName,
				Value: "1",
			},
			services:    []Service{},
			expectedErr: errNoServices,
		},
		"cookie value out of bounds": {
			cookie: &http.Cookie{
				Name:  lb.cookieName,
				Value: "5",
			},
			services: []Service{
				{
					Health: health.Always(true),
					Router: NewRedirect(url.URL{Scheme: "http", Host: "example.com"}),
				},
			},
			expectedURL: "http://example.com",
		},
		"no cookie": {
			services: []Service{
				{
					Health: health.Always(true),
					Router: NewRedirect(url.URL{Scheme: "http", Host: "foo.example.com"}),
				},
				{
					Health: health.Always(true),
					Router: NewRedirect(url.URL{Scheme: "http", Host: "bar.example.com"}),
				},
			},
			expectedURL: "http://foo.example.com",
		},
		"with cookie": {
			cookie: &http.Cookie{
				Name:  lb.cookieName,
				Value: "1",
			},
			services: []Service{
				{
					Health: health.Always(true),
					Router: NewRedirect(url.URL{Scheme: "http", Host: "wrong.example.com"}),
				},
				{
					Health: health.Always(true),
					Router: NewRedirect(url.URL{Scheme: "https", Host: "example.com"}),
				},
			},
			expectedURL: "https://example.com",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lb.services = test.services

			w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://example.com", nil)
			if test.cookie != nil {
				r.AddCookie(test.cookie)
			}

			url, err := lb.persistentService(w, r)
			if !errors.Is(err, test.expectedErr) {
				t.Fatalf("expected nil, got %v", err)
			}

			if err == nil && url.String() != test.expectedURL {
				t.Fatalf("expected %s, got %s", test.expectedURL, url.String())
			}
		})
	}
}

func TestLoadBalancerRoute(t *testing.T) {
	tests := map[string]struct {
		persistent  bool
		services    []Service
		expectedURL string
		expectedErr error
	}{
		"no services": {
			services:    []Service{},
			expectedErr: errNoServices,
		},
		"with services": {
			services: []Service{
				{
					Health: health.Always(true),
					Router: NewRedirect(url.URL{Scheme: "http", Host: "foo.example.com"}),
				},
				{
					Health: health.Always(true),
					Router: NewRedirect(url.URL{Scheme: "http", Host: "bar.example.com"}),
				},
			},
			expectedURL: "http://foo.example.com",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lb := NewLoadBalancer("host", &Failover{}, false, test.services)

			w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://example.com", nil)

			url, err := lb.Route(w, r)
			if !errors.Is(err, test.expectedErr) {
				t.Fatalf("expected nil, got %v", err)
			}

			if err == nil && url.String() != test.expectedURL {
				t.Fatalf("expected %s, got %s", test.expectedURL, url.String())
			}
		})
	}
}
