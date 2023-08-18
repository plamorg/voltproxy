package integration

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/plamorg/voltproxy/dockerapi"
)

func TestMain(m *testing.M) {
	// Silence log output.
	log.SetOutput(io.Discard)
	os.Exit(m.Run())
}

func TestSimpleHTTP(t *testing.T) {
	expectedCode := http.StatusAccepted
	server := NewMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(expectedCode)
	})
	conf := fmt.Sprintf(`
services:
  simple:
    host: foo.example.com
    redirect: "%s"`,
		server.URL())
	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHost("foo.example.com")
	defer res.Body.Close()

	if res.StatusCode != expectedCode {
		t.Fatalf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestSimpleHTTPS(t *testing.T) {
	expectedCode := http.StatusCreated
	server := NewMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(expectedCode)
	})
	conf := fmt.Sprintf(`
services:
  foo:
    host: secure.example.com
    tls: true
    redirect: "%s"`,
		server.URL())
	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHostTLS("secure.example.com")
	defer res.Body.Close()

	if res.StatusCode != expectedCode {
		t.Fatalf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestTLSNotAvailable(t *testing.T) {
	conf := `
services:
  notls:
    host: notls.example.com
    tls: false
    redirect: "invalid"`
	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHostTLS("notls.example.com")
	defer res.Body.Close()

	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status code %d, got %d", http.StatusNotFound, res.StatusCode)
	}
}

func TestForwardToCorrectService(t *testing.T) {
	expectedCode := http.StatusTeapot
	service3 := NewMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(expectedCode)
	})

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
    redirect: "invalid"`, service3.URL())
	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHost("service3.example.com")
	defer res.Body.Close()

	if res.StatusCode != http.StatusTeapot {
		t.Fatalf("expected status code %d, got %d", http.StatusTeapot, res.StatusCode)
	}
}

func TestServiceNotFound(t *testing.T) {
	conf := `
services:
  single:
    host: example.com
    redirect: "invalid"`
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

	expectedStatus := http.StatusNotFound
	if res.StatusCode != expectedStatus {
		t.Fatalf("expected status code %d, got %d", expectedStatus, res.StatusCode)
	}
}

func TestContainerFound(t *testing.T) {
	expectedCode := http.StatusTeapot
	server := NewMockServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(expectedCode)
	})
	host, port := server.SplitHostPort()

	containers := []dockerapi.Container{
		{
			Names: []string{"/oof", "/another-container"},
			Networks: map[string]dockerapi.IPAddress{
				"bar": "invalid",
			},
		},
		{
			Names: []string{"/bar", "/foo"},
			Networks: map[string]dockerapi.IPAddress{
				"bar": dockerapi.IPAddress(host),
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
      port: %s`, port)

	i := NewInstance(t, []byte(conf), containers)

	res := i.RequestHost("containerservice.example.com")
	defer res.Body.Close()

	if res.StatusCode != expectedCode {
		t.Fatalf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestMultipleMiddlewares(t *testing.T) {
	authServerRan := false

	fatalServer := NewMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("should not be called")
	})

	authServer := NewMockServer(t, (func(w http.ResponseWriter, r *http.Request) {
		authServerRan = true
		if r.Header.Get("X-Forwarded-For") == "" {
			t.Errorf("expected X-Forwarded-For header to be set")
		}
		if r.Header.Get("Custom-Header") != "test" {
			t.Errorf("expected header Custom-Header to have value test, got %s", r.Header.Get("Custom-Header"))
		}
		w.WriteHeader(http.StatusAccepted)
	}))

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
        - 0.0.0.0/32`, fatalServer.URL(), authServer.URL())
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
	serverName := "Server-Name"
	foo := NewMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(serverName, "foo")
	})
	bar := NewMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(serverName, "bar")
	})
	baz := NewMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(serverName, "baz")
	})

	conf := fmt.Sprintf(`
services:
    lb:
      host: lb.example.com
      loadBalancer:
        strategy: roundRobin
        serviceNames: ["foo", "bar", "baz"]
    foo:
      host: foo.example.com
      redirect: "%s"
    bar:
      redirect: "%s"
    baz:
      host: baz.example.com
      redirect: "%s"`, foo.URL(), bar.URL(), baz.URL())

	i := NewInstance(t, []byte(conf), nil)

	expected := []string{"foo", "bar", "baz", "foo", "bar", "baz"}

	for _, expectedServer := range expected {
		res := i.RequestHost("lb.example.com")
		defer res.Body.Close()

		if res.StatusCode != http.StatusOK {
			t.Fatalf("expected status code %d, got %d", http.StatusOK, res.StatusCode)
		}

		if res.Header.Get(serverName) != expectedServer {
			t.Fatalf("expected header %s to be %s, got %s", serverName, expectedServer, res.Header.Get(serverName))
		}
	}
}

func TestLoadBalancerPersistent(t *testing.T) {
	serverName := "Server-Name"
	foo := NewMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(serverName, "foo")
	})
	bar := NewMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(serverName, "bar")
	})

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
    redirect: "%s"`, foo.URL(), bar.URL())

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

	if res1.Header.Get(serverName) != "foo" || res2.Header.Get(serverName) != "foo" {
		t.Fatalf("expected both requests to be routed to foo, got %s and %s",
			res1.Header.Get(serverName), res2.Header.Get(serverName))
	}
}

func TestLoadBalancerHealth(t *testing.T) {
	unhealthy := NewMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	expectedCode := http.StatusTeapot
	up := NewMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(expectedCode)
	})

	conf := fmt.Sprintf(`
services:
  lb:
    host: lb.example.com
    loadBalancer:
      strategy: roundRobin
      serviceNames: ["unhealthy", "up"]
  unhealthy:
    redirect: "%s"
    health:
      path: "/health"
      interval: 0.5ms
  up:
    redirect: "%s"`, unhealthy.URL(), up.URL())

	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHost("lb.example.com")
	defer res.Body.Close()

	if res.StatusCode != expectedCode {
		t.Fatalf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestLoadBalancerHealthUp(t *testing.T) {
	// Although the server normally returns StatusForbidden, it returns a healthy status.
	// Thus, the health check should pass fine.
	expectedCode := http.StatusForbidden
	up := NewMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/what_is_my_health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(expectedCode)
	})

	conf := fmt.Sprintf(`
services:
  lb:
    host: lb.example.com
    loadBalancer:
      serviceNames: ["up"]
  up:
    redirect: "%s"
    health:
      interval: 0.5ms
      path: "/what_is_my_health"`, up.URL())
	i := NewInstance(t, []byte(conf), nil)

	res := i.RequestHost("lb.example.com")
	defer res.Body.Close()

	if res.StatusCode != expectedCode {
		t.Fatalf("expected status code %d, got %d", expectedCode, res.StatusCode)
	}
}

func TestFetchRemoteDynamically(t *testing.T) {
	server := NewMockServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})
	host, port := server.SplitHostPort()

	container := dockerapi.Container{
		Names: []string{"/server"},
		Networks: map[string]dockerapi.IPAddress{
			"net": dockerapi.IPAddress(host),
		},
	}

	conf := fmt.Sprintf(`
services:
  server:
    host: server.example.com
    container:
      name: "/server"
      network: "net"
      port: %s`, port)

	i := NewInstance(t, []byte(conf),
		[]dockerapi.Container{container}, // Containers on first request.
		[]dockerapi.Container{},          // Containers on second request.
		[]dockerapi.Container{container}, // Containers on third request.
		[]dockerapi.Container{container}, // Containers on fourth request.
	)

	expected := []int{http.StatusAccepted, http.StatusNotFound, http.StatusAccepted, http.StatusAccepted}

	for _, expectedCode := range expected {
		res := i.RequestHost("server.example.com")
		defer res.Body.Close()

		if res.StatusCode != expectedCode {
			t.Fatalf("expected status code %d, got %d", expectedCode, res.StatusCode)
		}
	}
}
