package middlewares

import (
	"net/http"
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
}

// NewAuthForward creates a new AuthForward middleware.
func NewAuthForward(address string, requestHeaders, responseHeaders []string) *AuthForward {
	return &AuthForward{
		Address:         address,
		RequestHeaders:  requestHeaders,
		ResponseHeaders: responseHeaders,
	}
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
			for key, values := range r.Header {
				authReq.Header[key] = append(authReq.Header[key], values...)
			}
		} else {
			for _, header := range a.RequestHeaders {
				authReq.Header.Set(header, r.Header.Get(header))
			}
		}

		resp, err := http.DefaultClient.Do(authReq)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		authFailed := resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices
		if authFailed {
			w.WriteHeader(resp.StatusCode)
			return
		}

		for _, header := range a.ResponseHeaders {
			canonicalHeader := http.CanonicalHeaderKey(header)
			r.Header[canonicalHeader] = append(r.Header[canonicalHeader], resp.Header[canonicalHeader]...)
		}

		next.ServeHTTP(w, r)
	})
}
