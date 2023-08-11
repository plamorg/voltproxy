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

// AuthForward is a middleware that forwards the request to an authentication server and
// proxies to the service if the authentication is successful.
type AuthForward struct {
	// The address of the authentication server.
	Address string `yaml:"address"`

	// The headers to forward from the request to the authentication server.
	// If this is nil, all headers will be forwarded.
	RequestHeaders []string `yaml:"requestHeaders"`

	// The headers to forward from the authentication server to the service.
	ResponseHeaders []string `yaml:"responseHeaders"`

	// Specifies whether to forward X-Forwarded-* headers to the authentication server.
	XForwarded bool `yaml:"xForwarded"`
}

// Handle communicates with the authentication server and proxies to the service if the
// authentication is successful.
func (a *AuthForward) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default().With(
			slog.String("host", r.Host),
			slog.Any("authForward", a))

		// Forward the request to the authentication server.
		logger.Debug("Forwarding request to authentication server")
		authReq, err := http.NewRequestWithContext(r.Context(), http.MethodGet, a.Address, nil)
		if err != nil {
			logger.Warn("Failed to create request to authentication server", slog.Any("error", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if a.RequestHeaders == nil {
			// Forward all request headers if none are specified.
			for key := range r.Header {
				authReq.Header.Set(key, r.Header.Get(key))
			}
		} else {
			for _, header := range a.RequestHeaders {
				authReq.Header.Set(header, r.Header.Get(header))
			}
		}

		if a.XForwarded {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err == nil {
				authReq.Header.Set(xForwardedFor, host)
			}
			authReq.Header.Set(xForwardedMethod, r.Method)
			if r.TLS != nil {
				authReq.Header.Set(xForwardedProto, "https")
			} else {
				authReq.Header.Set(xForwardedProto, "http")
			}
			authReq.Header.Set(xForwardedHost, r.Host)
			authReq.Header.Set(xForwardedURI, r.RequestURI)
		} else {
			for _, header := range xForwardedHeaders {
				authReq.Header.Del(header)
			}
		}

		noRedirectClient := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		res, err := noRedirectClient.Do(authReq)
		if err != nil {
			logger.Warn("Failed to send authentication request", slog.Any("error", err), slog.Any("request", authReq))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer res.Body.Close()
		logger.Debug("Authentication server responded", slog.Any("response", res))

		authFailed := res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices

		// If initial authentication has failed, try redirect to the next location given
		// by the authentication server.
		if authFailed {
			logger.Debug("Initial authentication failed, attempting to redirect to next location")

			if location, err := res.Location(); err == nil && location.String() != "" {
				http.Redirect(w, r, location.String(), res.StatusCode)
			} else {
				w.WriteHeader(res.StatusCode)
			}
			return
		}

		for _, header := range a.ResponseHeaders {
			r.Header.Set(header, res.Header.Get(header))
		}

		next.ServeHTTP(w, r)
	})
}
