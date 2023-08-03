package middlewares

import (
	"net"
	"net/http"
)

var (
	xForwardedFor    = "X-Forwarded-For"
	xForwardedMethod = "X-Forwarded-Method"
	xForwardedProto  = "X-Forwarded-Proto"
	xForwardedHost   = "X-Forwarded-Host"
	xForwardedURI    = "X-Forwarded-Uri"
)

// AuthForward is a middleware that forwards the request to an authentication server and
// proxies to the service if the authentication is successful.
type AuthForward struct {
	Address string

	// The headers to forward from the request to the authentication server.
	// If this is nil, all headers will be forwarded.
	RequestHeaders []string

	// The headers to forward from the authentication server to the service.
	ResponseHeaders []string

	ForwardXForwarded bool
}

// Handle communicates with the authentication server and proxies to the service if the
// authentication is successful.
func (a *AuthForward) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Forward the request to the authentication server.
		authReq, err := http.NewRequest(http.MethodGet, a.Address, nil)
		if err != nil {
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

		if a.ForwardXForwarded {
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
			for _, header := range []string{xForwardedFor, xForwardedMethod, xForwardedProto, xForwardedHost, xForwardedURI} {
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
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer res.Body.Close()

		authFailed := res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices

		// If initial authentication has failed, try redirect to the next location given
		// by the authentication server.
		if authFailed {
			// Ensure that the original headers sent to the authentication server are once
			// again sent to the redirect location.
			for header := range res.Header {
				w.Header().Set(header, res.Header.Get(header))
			}

			w.WriteHeader(res.StatusCode)
			return
		}

		for _, header := range a.ResponseHeaders {
			r.Header.Set(header, res.Header.Get(header))
		}

		next.ServeHTTP(w, r)
	})
}
