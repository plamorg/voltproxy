// Package services provides a way to define services that can be proxied.
package services

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/plamorg/voltproxy/middlewares"
	"github.com/plamorg/voltproxy/services/health"
)

// Router is something that can route a request to a service.
type Router interface {
	Route(http.ResponseWriter, *http.Request) (*url.URL, error)
}

// Service is a service that can be proxied.
type Service struct {
	TLS         bool
	Middlewares []middlewares.Middleware
	Health      health.Checker

	Router Router
}

// LaunchHealthChecks starts the health checks for all services.
func LaunchHealthChecks(services map[string]Service) {
	for host, service := range services {
		// This is a workaround for the loop variable problem.
		// See: https://github.com/golang/go/wiki/LoopvarExperiment
		service := service

		logger := slog.Default().With(slog.String("host", host), slog.Any("service", service))

		go service.Health.Launch(service.Router.Route)
		go func() {
			for {
				res := <-service.Health.Check()
				if res.Err != nil || !res.Up {
					logger.Warn("Failed health check", slog.Any("result", res))
				} else {
					logger.Debug("Successful Health check", slog.Any("result", res))
				}
			}
		}()
	}
}

// Handler returns a http.Handler that proxies requests to services, redirecting to TLS if applicable.
func Handler(services map[string]Service) http.Handler {
	return handler(services, false)
}

// TLSHandler returns a http.Handler that proxies requests to services with TLS enabled.
func TLSHandler(services map[string]Service) http.Handler {
	return handler(services, true)
}

func handler(services map[string]Service, tls bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default().With(slog.String("host", r.Host), slog.Bool("tls", tls))

		logger.Debug("Handling request")

		service, ok := services[r.Host]
		if !ok {
			logger.Debug("No service found for host")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		logger = logger.With(slog.Any("service", service))

		if service.TLS && !tls {
			redirectURL := "https://" + r.Host + r.URL.String()
			logger.Debug("Redirecting to TLS server", slog.String("redirect", redirectURL))
			http.Redirect(w, r, redirectURL, http.StatusMovedPermanently)
			return
		}
		if !service.TLS && tls {
			logger.Debug("Service does not support TLS")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		route, err := service.Router.Route(w, r)
		if err != nil {
			logger.Warn("Error while routing to service", slog.Any("error", err))
			status := http.StatusInternalServerError
			w.WriteHeader(status)
			return
		}
		logger = logger.With(slog.String("route", route.String()))

		var handler http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proxy := httputil.NewSingleHostReverseProxy(route)
			r.Host = route.Host
			logger.Debug("Proxying request")
			proxy.ServeHTTP(w, r)
		})

		middlewares := service.Middlewares
		if len(middlewares) > 0 {
			for _, middleware := range middlewares {
				handler = middleware.Handle(handler)
			}
		}

		handler.ServeHTTP(w, r)
	})
}
