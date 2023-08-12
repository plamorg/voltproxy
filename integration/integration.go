// Package integration provides an interface to write integration tests for the reverse proxy.
package integration

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/plamorg/voltproxy/config"
	"github.com/plamorg/voltproxy/dockerapi"
	"github.com/plamorg/voltproxy/services"
)

// TeapotServer is a test server that always returns http.StatusTeapot.
var TeapotServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusTeapot)
}))

// Instance is a test instance of the reverse proxy corresponding to a config.
type Instance struct {
	services services.List
	docker   *dockerapi.Mock
	t        *testing.T
	url      string
	tlsURL   string
}

// NewInstance creates a new instance of the reverse proxy with the given config.
func NewInstance(t *testing.T, confData []byte, containers []types.Container) *Instance {
	t.Helper()
	conf, err := config.Parse(confData)
	if err != nil {
		t.Fatal(err)
	}

	docker := dockerapi.NewMock(containers)

	services, err := conf.ServiceList(docker)
	if err != nil {
		t.Fatal(err)
	}

	err = services.StartHealthChecks()
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(services.Handler())
	tlsServer := httptest.NewServer(services.TLSHandler())
	t.Cleanup(func() {
		server.Close()
		tlsServer.Close()
	})

	return &Instance{
		services: services,
		docker:   docker,
		t:        t,
		url:      server.URL,
		tlsURL:   tlsServer.URL,
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
