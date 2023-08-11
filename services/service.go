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

// Config describes the user defined configuration for a service.
type Config struct {
	Host        string                   `yaml:"host"`
	TLS         bool                     `yaml:"tls"`
	Middlewares *middlewares.Middlewares `yaml:"middlewares"`
}

func (c *Config) data() data {
	var middlewares []middlewares.Middleware
	if c.Middlewares != nil {
		middlewares = c.Middlewares.List()
	}
	return data{
		host:        c.Host,
		tls:         c.TLS,
		middlewares: middlewares,
	}
}

type data struct {
	host        string
	tls         bool
	middlewares []middlewares.Middleware
}

type Service interface {
	Data() data
	Remote() (*url.URL, error)
}

// List is a list of services which can be used to proxy requests (http.Request).
type List []Service

func (l *List) findServiceWithHost(host string) (*Service, error) {
	for _, service := range *l {
		if service.Data().host == host {
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

		if (*service).Data().tls && !tls {
			redirectURL := "https://" + r.Host + r.URL.String()
			logger.Debug("Redirecting to TLS server", slog.String("redirect", redirectURL))
			http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
			return
		}
		if !(*service).Data().tls && tls {
			logger.Debug("Service does not support TLS")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		remote, err := (*service).Remote()
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

		middlewares := (*service).Data().middlewares
		if len(middlewares) > 0 {
			slog.Debug("Adding middlewares", slog.Int("count", len(middlewares)))
			for _, middleware := range middlewares {
				handler = middleware.Handle(handler)
			}
		}

		handler.ServeHTTP(w, r)
	})
}
