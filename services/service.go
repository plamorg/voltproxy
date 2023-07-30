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

type service interface {
	Host() string
	Remote() (*url.URL, error)
	Middlewares() []middlewares.Middleware
}

// List is a list of services which can be used to proxy requests (http.Request).
type List []service

func (l *List) findServiceWithHost(host string) (*service, error) {
	for _, service := range *l {
		if service.Host() == host {
			return &service, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNoServiceFound, host)
}

// Proxy proxies requests to an appropriate service based on the request's host.
func (l *List) Proxy(r *http.Request, w http.ResponseWriter) {
	service, err := l.findServiceWithHost(r.Host)
	if err != nil {
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

	middlewares := (*service).Middlewares()
	for _, middleware := range middlewares {
		handler = middleware.Handle(proxy)
	}

	handler.ServeHTTP(w, r)
}
