package services

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

var errNoServices = fmt.Errorf("no services in pool")

const (
	lbCookiePrefix     = "voltproxy-lb-"
	lbCookieNameLength = 8
	lbCookieBase       = 10
	lbCookieBitSize    = 64
)

// LoadBalancer is a service that load balances between other services.
type LoadBalancer struct {
	cookieName string

	strategy   Strategy
	persistent bool
	services   []*Service
}

func generateCookieName(host string) string {
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%s%s", lbCookiePrefix, host)))

	return fmt.Sprintf("%x", hash.Sum(nil)[:lbCookieNameLength])
}

// NewLoadBalancer creates a new load balancer service.
func NewLoadBalancer(host string, strategy Strategy, persistent bool) *LoadBalancer {
	return &LoadBalancer{
		cookieName: generateCookieName(host),
		strategy:   strategy,
		persistent: persistent,
	}
}

// SetServices sets the services that the load balancer will load balance.
func (l *LoadBalancer) SetServices(services []*Service) {
	l.services = services
}

func (l *LoadBalancer) persistentService(w http.ResponseWriter, r *http.Request) (*url.URL, error) {
	if len(l.services) == 0 {
		return nil, errNoServices
	}
	if cookie, err := r.Cookie(l.cookieName); err == nil {
		cookieNext, err := strconv.ParseUint(cookie.Value, lbCookieBase, lbCookieBitSize)
		validCookie := err == nil && cookieNext < uint64(len(l.services))

		if validCookie && l.services[cookieNext].Health.Up() {
			return l.services[cookieNext].Router.Route(w, r)
		}
	}

	next := l.strategy.Select(l.services, r)
	cookie := &http.Cookie{
		Name:     l.cookieName,
		Value:    fmt.Sprint(next),
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
	return l.services[next].Router.Route(w, r)
}

// Route returns the remote URL of the next service in the load balancer.
func (l *LoadBalancer) Route(w http.ResponseWriter, r *http.Request) (*url.URL, error) {
	if len(l.services) == 0 {
		return nil, errNoServices
	}
	if l.persistent {
		return l.persistentService(w, r)
	}
	next := l.strategy.Select(l.services, r)
	return l.services[next].Router.Route(w, r)
}

var _ Router = (*LoadBalancer)(nil)
