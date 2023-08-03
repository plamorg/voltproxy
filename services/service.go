// Package services provides a way to define services that can be proxied.
package services

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/plamorg/voltproxy/middlewares"
)

// errNoServiceFound is returned when no service with the host is found.
var errNoServiceFound = fmt.Errorf("no service with host")

// Config describes the user defined configuration for a service.
type Config struct {
	Host        string
	TLS         bool
	Middlewares []middlewares.Middleware
}

type service interface {
	Config() Config
	Remote() (*url.URL, error)
}

// List is a list of services which can be used to proxy requests (http.Request).
type List []service

func (l *List) findServiceWithHost(host string) (*service, error) {
	for _, service := range *l {
		if service.Config().Host == host {
			return &service, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", errNoServiceFound, host)
}

// Handler returns a http.Handler that proxies requests to services, redirecting to TLS if applicable.
func (l *List) Handler() http.Handler {
	return l.handler(false)
}

// TLSHandler returns a http.Handler that proxies requests to services with TLS enabled.
func (l *List) TLSHandler() http.Handler {
	return l.handler(true)
}

func (l *List) handler(tls bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service, err := l.findServiceWithHost(r.Host)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if (*service).Config().TLS && !tls {
			http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
			return
		}
		if !(*service).Config().TLS && tls {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		remote, err := (*service).Remote()
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxy := httputil.NewSingleHostReverseProxy(remote)
			r.Host = remote.Host
			proxy.ServeHTTP(w, r)
		})

		middlewares := (*service).Config().Middlewares
		for _, middleware := range middlewares {
			handler = middleware.Handle(handler)
		}

		handler.ServeHTTP(w, r)
	})
}
