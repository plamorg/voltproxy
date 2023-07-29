package middlewares

import (
	"net"
	"net/http"
)

// IPAllow is a middleware that checks the remote address of the request and
// compares it to the allowed IP addresses.
// If the remote address is not in the allowed IP addresses, it returns a 403 Forbidden.
type IPAllow struct {
	AllowedIPs []string
}

// NewIPAllow returns a new IPAllow middleware.
func NewIPAllow(allowedIPs []string) *IPAllow {
	return &IPAllow{allowedIPs}
}

// Handle checks the remote address of the request and compares it to the allowed IP addresses.
func (ip *IPAllow) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			for _, allowedIP := range ip.AllowedIPs {
				if host == allowedIP {
					next.ServeHTTP(w, r)
					return
				}
			}
		}
		w.WriteHeader(http.StatusForbidden)
	})
}
