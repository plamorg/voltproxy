package integration

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
)

func TestSimpleHTTP(t *testing.T) {
	conf := fmt.Sprintf(`
services:
  simple:
    host: foo.example.com
    redirect: "%s"`,
		TeapotServer.URL)
	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHost("foo.example.com")
	defer res.Body.Close()

	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status code %d, got %d", http.StatusTeapot, res.StatusCode)
	}
}

func TestSimpleHTTPS(t *testing.T) {
	conf := fmt.Sprintf(`
services:
  foo:
    host: secure.example.com
    tls: true
    redirect: "%s"`,
		TeapotServer.URL)
	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHostTLS("secure.example.com")
	defer res.Body.Close()

	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status code %d, got %d", http.StatusTeapot, res.StatusCode)
	}
}

func TestTLSNotAvailable(t *testing.T) {
	conf := fmt.Sprintf(`
services:
  notls:
    host: notls.example.com
    tls: false
    redirect: "%s"`, TeapotServer.URL)
	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHostTLS("notls.example.com")
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, res.StatusCode)
	}
}

func TestForwardToCorrectService(t *testing.T) {
	conf := fmt.Sprintf(`
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
    redirect: "invalid"`, TeapotServer.URL)
	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHost("service3.example.com")
	defer res.Body.Close()

	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status code %d, got %d", http.StatusTeapot, res.StatusCode)
	}
}

func TestServiceNotFound(t *testing.T) {
	conf := fmt.Sprintf(`
services:
  single:
    host: example.com
    redirect: "%s"`, TeapotServer.URL)
	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHost("notfound.example.com")
	defer res.Body.Close()
	resTLS := i.RequestHostTLS("notfound.example.com")
	defer resTLS.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, res.StatusCode)
	}

	if resTLS.StatusCode != http.StatusNotFound {
		t.Fatalf("expected TLS status code %d, got %d", http.StatusNotFound, resTLS.StatusCode)
	}
}

func TestContainerNotFound(t *testing.T) {
	conf := `
services:
  container:
    host: container.example.com
    container:
      name: "/container"
      network: "host"
      port: 80`
	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHost("container.example.com")
	defer res.Body.Close()

	if res.StatusCode != http.StatusInternalServerError {
		t.Fatalf("expected status code %d, got %d", http.StatusInternalServerError, res.StatusCode)
	}
}

func TestContainerFound(t *testing.T) {
	teapotHost, teapotPort, err := net.SplitHostPort(strings.TrimPrefix(TeapotServer.URL, "http://"))
	if err != nil {
		t.Fatalf("failed to parse host and port from %s", TeapotServer.URL)
	}

	containers := []types.Container{
		{
			Names: []string{"/oof", "/another-container"},
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"bar": {
						IPAddress: "invalid",
					},
				},
			},
		},
		{
			Names: []string{"/bar", "/foo"},
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"bar": {
						IPAddress: teapotHost,
					},
				},
			},
		},
	}
	conf := fmt.Sprintf(`
services:
  containerservice:
    host: containerservice.example.com
    container:
      name: "/foo"
      network: "bar"
      port: %s`, teapotPort)

	i := NewInstance(t, []byte(conf), containers)

	res := i.RequestHost("containerservice.example.com")
	defer res.Body.Close()

	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status code %d, got %d", http.StatusTeapot, res.StatusCode)
	}
}

func TestMultipleMiddlewares(t *testing.T) {
	authServerRan := false

	fatalServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	}))

	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authServerRan = true
		if r.Header.Get("X-Forwarded-For") == "" {
			t.Errorf("expected X-Forwarded-For header to be set")
		}
		if r.Header.Get("Custom-Header") != "test" {
			t.Errorf("expected header Custom-Header to have value test, got %s", r.Header.Get("Custom-Header"))
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer authServer.Close()

	conf := fmt.Sprintf(`
services:
  middle:
    host: example.com
    redirect: "%s"
    middlewares:
      authForward:
        address: "%s"
        xForwarded: true
        requestHeaders: ["Custom-Header"]
      ipAllow:
        - 0.0.0.0/32`, fatalServer.URL, authServer.URL)
	// In CIDR notation, 0.0.0.0/0 represents all IPv4 addresses.
	// This means the ipallow middleware will allow all requests regardless of incoming IP address.

	i := NewInstance(t, []byte(conf), nil)

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, i.URL(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "example.com"
	req.Header.Set("Custom-Header", "test")

	res := i.Request(req)
	defer res.Body.Close()

	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("expected status code %d, got %d", http.StatusOK, res.StatusCode)
	}

	if !authServerRan {
		t.Fatalf("expected auth server to run")
	}
}

func TestLoadBalancerRoundRobin(t *testing.T) {
	fooServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	barServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	conf := fmt.Sprintf(`
services:
    lb:
      host: lb.example.com
      loadBalancer:
        strategy: roundRobin
        serviceNames: ["foo", "bar", "invalid"]
    foo:
      host: foo.example.com
      redirect: "%s"
    bar:
      redirect: "%s"
    invalid:
      host: invalid.example.com
      redirect: "invalid"`, fooServer.URL, barServer.URL)

	i := NewInstance(t, []byte(conf), nil)

	fooRes1 := i.RequestHost("lb.example.com")
	defer fooRes1.Body.Close()
	barRes := i.RequestHost("lb.example.com")
	defer barRes.Body.Close()

	invalidRes := i.RequestHost("lb.example.com")
	defer invalidRes.Body.Close()

	fooRes2 := i.RequestHost("lb.example.com")
	defer fooRes2.Body.Close()

	if fooRes1.StatusCode != http.StatusCreated {
		t.Fatalf("expected status code %d, got %d", http.StatusCreated, fooRes1.StatusCode)
	}

	if barRes.StatusCode != http.StatusAccepted {
		t.Fatalf("expected status code %d, got %d", http.StatusAccepted, barRes.StatusCode)
	}

	if fooRes2.StatusCode != http.StatusCreated {
		t.Fatalf("expected status code %d, got %d", http.StatusCreated, fooRes2.StatusCode)
	}
}

func TestLoadBalancerPersistent(t *testing.T) {
	barServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	conf := fmt.Sprintf(`
services:
  lb:
    host: lb.example.com
    loadBalancer:
      persistent: true
      strategy: roundRobin
      serviceNames: ["foo", "bar"]
  foo:
    redirect: "%s"
  bar:
    redirect: "%s"`, TeapotServer.URL, barServer.URL)

	i := NewInstance(t, []byte(conf), nil)

	res1 := i.RequestHost("lb.example.com")
	defer res1.Body.Close()

	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, i.URL(), nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "lb.example.com"
	for _, cookie := range res1.Cookies() {
		req.AddCookie(cookie)
	}
	res2 := i.Request(req)
	defer res2.Body.Close()

	if res1.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status code %d, got %d", http.StatusTeapot, res1.StatusCode)
	}

	if res2.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status code %d, got %d", http.StatusTeapot, res2.StatusCode)
	}
}

func TestLoadBalancerHealth(t *testing.T) {
	down := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))

	conf := fmt.Sprintf(`
services:
  lb:
    host: lb.example.com
    loadBalancer:
      strategy: roundRobin
      serviceNames: ["down", "up"]
  down:
    redirect: "%s"
    health:
      path: "/health"
      interval: 0.5ms
  up:
    redirect: "%s"`, down.URL, TeapotServer.URL)

	i := NewInstance(t, []byte(conf), nil)

	ticker := time.NewTicker(2 * time.Millisecond)
	defer ticker.Stop()
	<-ticker.C

	res := i.RequestHost("lb.example.com")
	defer res.Body.Close()

	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status code %d, got %d", http.StatusTeapot, res.StatusCode)
	}
}
