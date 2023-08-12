// Package services provides a way to define services that can be proxied.
package services

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
	"reflect"

	"github.com/plamorg/voltproxy/middlewares"
)

// Services is a structure of all services configurations.
type Services struct {
	Container    *ContainerInfo    `yaml:"container"`
	Redirect     string            `yaml:"redirect"`
	LoadBalancer *LoadBalancerInfo `yaml:"loadBalancer"`
}

// Validate returns true if there is exactly one service.
func (s *Services) Validate() bool {
	v := reflect.ValueOf(*s)
	count := 0
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsZero() {
			continue
		}
		if count > 0 {
			return false
		}
		count++
	}
	return count == 1
}

// errNoServiceFound is returned when no service with the host is found.
var errNoServiceFound = fmt.Errorf("no service with host")

// Data is the data required to create a service.
type Data struct {
	Host        string
	TLS         bool
	Middlewares []middlewares.Middleware
}

// NewData creates a new service Data by calling the middlewares List method.
func NewData(host string, tls bool, m *middlewares.Middlewares) Data {
	var l []middlewares.Middleware
	if m != nil {
		l = m.List()
	}
	return Data{
		Host:        host,
		TLS:         tls,
		Middlewares: l,
	}
}

// Service is an interface describing an arbitrary service that can be proxied.
type Service interface {
	Data() Data
	Remote(http.ResponseWriter, *http.Request) (*url.URL, error)
}

// List is a list of services which can be used to proxy requests (http.Request).
type List []Service

func (l *List) findServiceWithHost(host string) (*Service, error) {
	for _, service := range *l {
		if service.Data().Host == host {
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
		logger := slog.Default().With(slog.String("host", r.Host), slog.Bool("tls", tls))

		logger.Debug("Handling request")
		service, err := l.findServiceWithHost(r.Host)
		if err != nil {
			logger.Debug("Error while finding service", slog.Any("error", err))
			w.WriteHeader(http.StatusNotFound)
			return
		}

		if (*service).Data().TLS && !tls {
			redirectURL := "https://" + r.Host + r.URL.String()
			logger.Debug("Redirecting to TLS server", slog.String("redirect", redirectURL))
			http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
			return
		}
		if !(*service).Data().TLS && tls {
			logger.Debug("Service does not support TLS")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		remote, err := (*service).Remote(w, r)
		if err != nil {
			logger.Warn("Error while getting remote URL", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxy := httputil.NewSingleHostReverseProxy(remote)
			r.Host = remote.Host
			logger.Debug("Serving remote", slog.Any("remote", remote))
			proxy.ServeHTTP(w, r)
		})

		middlewares := (*service).Data().Middlewares
		if len(middlewares) > 0 {
			slog.Debug("Adding middlewares", slog.Int("count", len(middlewares)))
			for _, middleware := range middlewares {
				handler = middleware.Handle(handler)
			}
		}

		handler.ServeHTTP(w, r)
	})
}
