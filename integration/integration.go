// Package integration provides an interface to write integration tests for the reverse proxy.
package integration

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/plamorg/voltproxy/config"
	"github.com/plamorg/voltproxy/dockerapi"
	"github.com/plamorg/voltproxy/services"
)

// MockServer is a test instance of a server.
type MockServer struct {
	t      *testing.T
	server *httptest.Server
}

// NewMockServer creates a new test instance of a server.
// The server will be closed when the test function returns.
func NewMockServer(t *testing.T, handle http.HandlerFunc) *MockServer {
	t.Helper()
	server := httptest.NewServer(handle)
	t.Cleanup(func() {
		server.Close()
	})
	return &MockServer{t, server}
}

// URL returns the URL of the server.
func (s *MockServer) URL() string {
	return s.server.URL
}

// SplitHostPort returns the host and port of the server.
// Will call t.Fatal if the URL is invalid.
func (s *MockServer) SplitHostPort() (host, port string) {
	s.t.Helper()
	host, port, err := net.SplitHostPort(strings.TrimPrefix(s.server.URL, "http://"))
	if err != nil {
		s.t.Fatal(err)
	}
	return
}

// Instance is a test instance of the reverse proxy corresponding to a config.
type Instance struct {
	serviceMap map[string]services.Service
	docker     *dockerapi.Mock
	t          *testing.T
	url        string
	tlsURL     string
}

// NewInstance creates a new instance of the reverse proxy with the given config.
func NewInstance(t *testing.T, confData []byte, containers ...[]dockerapi.Container) *Instance {
	t.Helper()
	conf, err := config.Parse(confData)
	if err != nil {
		t.Fatal(err)
	}

	docker := dockerapi.NewMock(containers...)

	serviceMap, err := conf.ServiceMap(docker)
	if err != nil {
		t.Fatal(err)
	}

	services.LaunchHealthChecks(serviceMap)

	server := httptest.NewServer(services.Handler(serviceMap))
	tlsServer := httptest.NewServer(services.TLSHandler(serviceMap))
	t.Cleanup(func() {
		server.Close()
		tlsServer.Close()
	})

	return &Instance{
		serviceMap: serviceMap,
		docker:     docker,
		t:          t,
		url:        server.URL,
		tlsURL:     tlsServer.URL,
	}
}

// URL returns the URL of the reverse proxy instance.
func (i *Instance) URL() string {
	return i.url
}

// TLSURL returns the TLS URL of the reverse proxy instance.
func (i *Instance) TLSURL() string {
	return i.tlsURL
}

// RequestHost sends a request to the reverse proxy with the given host.
func (i *Instance) RequestHost(host string) *http.Response {
	i.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, i.url, nil)
	if err != nil {
		i.t.Fatal(err)
	}
	req.Host = host
	return i.Request(req)
}

// RequestHostTLS sends a request to the reverse proxy with the given host, using TLS.
func (i *Instance) RequestHostTLS(host string) *http.Response {
	i.t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, i.tlsURL, nil)
	if err != nil {
		i.t.Fatal(err)
	}
	req.Host = host
	return i.Request(req)
}

// Request sends a request to the reverse proxy.
func (i *Instance) Request(req *http.Request) *http.Response {
	i.t.Helper()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		i.t.Fatal(err)
	}

	return res
}
