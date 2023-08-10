package middlewares

import (
	"log/slog"
	"net"
	"net/http"
)

// IPAllow is a middleware that only allows requests from a list of IP addresses.
// Accepts IPs in CIDR notation.
type IPAllow []string

// NewIPAllow creates a new IPAllow middleware.
func NewIPAllow(allowedIPs []string) *IPAllow {
	return (*IPAllow)(&allowedIPs)
}

// Handle checks the remote address of the request and compares it to the allowed IP addresses.
// If the remote address is not in the allowed IP addresses, it returns a 403 Forbidden.
func (ip *IPAllow) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logger := slog.Default().With(
			slog.String("host", r.Host),
			slog.Group("ipallow",
				slog.Any("allowedIPs", *ip),
				slog.String("remoteAddr", r.RemoteAddr),
			))

		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			for _, allowedIP := range *ip {
				if host == allowedIP || inCIDR(host, allowedIP) {
					logger.Debug("Remote address is allowed")
					next.ServeHTTP(w, r)
					return
				}
			}
		} else {
			logger.Warn("Error while splitting remote address", slog.Any("error", err))
		}
		logger.Debug("Remote address is not allowed")
		w.WriteHeader(http.StatusForbidden)
	})
}

func inCIDR(ip string, cidr string) bool {
	_, ipnet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false
	}
	return ipnet.Contains(net.ParseIP(ip))
}
