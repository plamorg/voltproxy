package integration

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSimpleHTTP(t *testing.T) {
	i := NewInstance(t, []byte(fmt.Sprintf(`
services:
  simple:
    host: foo.example.com
    redirect: "%s"`, TeapotServer.URL)))

	res := i.RequestHost("foo.example.com")

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

	res := i.RequestHostTLS("secure.example.com")

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

	res := i.RequestHost("example.com")
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

	res := i.RequestHostTLS("notls.example.com")

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

	res := i.RequestHost("service3.example.com")

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

	res := i.RequestHost("notfound.example.com")
	resTLS := i.RequestHostTLS("notfound.example.com")

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

	res := i.RequestHost("container.example.com")

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, res.StatusCode)
	}
}

func TestMultipleMiddlewares(t *testing.T) {
	headerForwarder := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Custom-Header", r.Header.Get("Custom-Header"))
		w.WriteHeader(http.StatusOK)
	}))

	// headerForwarder is used as both the destination service and the authforward authentication server.
	i := NewInstance(t, []byte(fmt.Sprintf(`
services:
  middle:
    host: example.com
    redirect: "%s"
    middlewares:
      authforward:
        address: "%s"
        requestheaders: ["Custom-Header"]
        responseheaders: ["Custom-Header"]
      ipallow:
        - 0.0.0.0/0`, headerForwarder.URL, headerForwarder.URL)))
	// In CIDR notation, 0.0.0.0/0 represents all IPv4 addresses.
	// This means that the ipallow middleware will allow all requests regardless of incoming IP address.

	req, err := http.NewRequest("GET", i.URL(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.com"
	req.Header.Set("Custom-Header", "test")

	res := i.Request(req)

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	if res.Header.Get("Custom-Header") != "test" {
		t.Fatalf("expected header value %s, got %s", "test", res.Header.Get("Custom-Header"))
	}
}
