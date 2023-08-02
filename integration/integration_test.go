package integration

import (
	"fmt"
	"net/http"
	"testing"
)

func TestSimpleHTTP(t *testing.T) {
	i := NewInstance(t, []byte(fmt.Sprintf(`
services:
  simple:
    host: foo.example.com
    redirect: "%s"`, TeapotServer.URL)))

	res := i.Request("foo.example.com")

	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status code %d, got %d", http.StatusTeapot, res.StatusCode)
	}
}

func TestSimpleHTTPS(t *testing.T) {
	i := NewInstance(t, []byte(fmt.Sprintf(`
services:
  foo:
    host: secure.example.com
    tls: true
    redirect: "%s"`, TeapotServer.URL)))

	res := i.RequestTLS("secure.example.com")

	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status code %d, got %d", http.StatusTeapot, res.StatusCode)
	}
}

func TestRedirectToTLS(t *testing.T) {
	i := NewInstance(t, []byte(fmt.Sprintf(`
services:
  service:
    host: example.com
    tls: true
    redirect: "%s"`, TeapotServer.URL)))

	expectedURL := "https://example.com/"

	res := i.Request("example.com")
	url := res.Request.URL.String()
	if url != expectedURL {
		t.Fatalf("expected url %s, got %s", expectedURL, url)
	}
}

func TestTLSNotAvailable(t *testing.T) {
	i := NewInstance(t, []byte(fmt.Sprintf(`
services:
  notls:
    host: notls.example.com
    tls: false
    redirect: "%s"`, TeapotServer.URL)))

	res := i.RequestTLS("notls.example.com")

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, res.StatusCode)
	}
}

func TestForwardToCorrectService(t *testing.T) {
	i := NewInstance(t, []byte(fmt.Sprintf(`
services:
  service1:
    host: service1.example.com
    redirect: "invalid"
  service2:
    host: service2.example.com
    redirect: "invalid"
  service3:
    host: service3.example.com
    redirect: "%s"
  service4:
    host: service4.example.com
    redirect: "invalid"`, TeapotServer.URL)))

	res := i.Request("service3.example.com")

	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status code %d, got %d", http.StatusTeapot, res.StatusCode)
	}
}

func TestServiceNotFound(t *testing.T) {
	i := NewInstance(t, []byte(fmt.Sprintf(`
services:
  single:
    host: example.com
    redirect: "%s"`, TeapotServer.URL)))

	res := i.Request("notfound.example.com")
	resTLS := i.RequestTLS("notfound.example.com")

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, res.StatusCode)
	}

	if resTLS.StatusCode != http.StatusNotFound {
		t.Fatalf("expected TLS status code %d, got %d", http.StatusNotFound, resTLS.StatusCode)
	}
}

func TestNoContainerFound(t *testing.T) {
	i := NewInstance(t, []byte(fmt.Sprintf(`
services:
  container:
    host: container.example.com
    container:
      name: "/container"
      network: "host"
      port: 80`)))

	res := i.Request("container.example.com")

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, res.StatusCode)
	}
}

func TestWithMiddleware(t *testing.T) {
	i := NewInstance(t, []byte(fmt.Sprintf(`
services:
  service:
    host: foo.example.com
    redirect: "%s"
    middlewares:
      ipallow:
        - 10.9.0.0/16`, TeapotServer.URL)))

	res := i.Request("foo.example.com")

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status code %d, got %d", http.StatusForbidden, res.StatusCode)
	}
}
