package middlewares

import (
	"log/slog"
	"net"
	"net/http"
)

const (
	xForwardedFor    = "X-Forwarded-For"
	xForwardedMethod = "X-Forwarded-Method"
	xForwardedProto  = "X-Forwarded-Proto"
	xForwardedHost   = "X-Forwarded-Host"
	xForwardedURI    = "X-Forwarded-Uri"
)

var xForwardedHeaders = []string{
	xForwardedFor,
	xForwardedMethod,
	xForwardedProto,
	xForwardedHost,
	xForwardedURI,
}

// XForward is a middleware that adds X-Forwarded headers to the request.
type XForward struct {
	Enable bool `yaml:"enable"`
}

func xForward(newReq *http.Request, r http.Request) {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		newReq.Header.Set(xForwardedFor, host)
	}
	newReq.Header.Set(xForwardedMethod, r.Method)
	if r.TLS != nil {
		newReq.Header.Set(xForwardedProto, "https")
	} else {
		newReq.Header.Set(xForwardedProto, "http")
	}
	newReq.Header.Set(xForwardedHost, r.Host)
	newReq.Header.Set(xForwardedURI, r.RequestURI)
}

// Handle adds X-Forwarded headers to the request.
func (x *XForward) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default().With(
			slog.String("host", r.Host),
			slog.Any("xForward", x))

		if !x.Enable {
			next.ServeHTTP(w, r)
			return
		}

		xForward(r, *r)

		logger.Debug("Added X-Forwarded headers")

		next.ServeHTTP(w, r)
	})
}
