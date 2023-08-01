// Package services provides a way to define services that can be proxied.
package services

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/plamorg/voltproxy/middlewares"
)

// ErrNoServiceFound is returned when no service with the host is found.
var ErrNoServiceFound = fmt.Errorf("no service with host")

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
	return nil, fmt.Errorf("%w: %s", ErrNoServiceFound, host)
}

// Proxy creates a proxy handler based on whether TLS is enabled or not.
func (l *List) Proxy(tls bool) http.Handler {
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
		if errors.Is(err, ErrNoServiceFound) {
			w.WriteHeader(http.StatusNotFound)
			return
		} else if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		r.Host = remote.Host
		proxy := httputil.NewSingleHostReverseProxy(remote)
		var handler http.Handler = proxy

		middlewares := (*service).Config().Middlewares
		for _, middleware := range middlewares {
			handler = middleware.Handle(proxy)
		}

		handler.ServeHTTP(w, r)
	})
}
