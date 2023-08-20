package services

import (
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
	expectedURL := "https://example.com"
	lb := NewLoadBalancer("host", &Failover{}, true, []Service{
		{Router: NewRedirect(url.URL{Scheme: "http", Host: "wrong.example.com"})},
		{
			Health: health.Always(true),
			Router: NewRedirect(url.URL{Scheme: "https", Host: "example.com"}),
		},
	})

	w, r := httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	r.AddCookie(&http.Cookie{
		Name:  lb.cookieName,
		Value: "1",
	})

	url, err := lb.persistentService(w, r)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}

	if url.String() != expectedURL {
		t.Fatalf("expected %s, got %s", expectedURL, url.String())
	}
}
