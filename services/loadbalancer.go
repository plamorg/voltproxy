package services

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/plamorg/voltproxy/services/selection"
)

const (
	lbCookiePrefix     = "voltproxy-lb-"
	lbCookieNameLength = 8
	lbCookieBase       = 10
	lbCookieBitSize    = 64
)

var (
	// ErrInvalidStrategy is returned when an unexpected strategy string is specified.
	errInvalidStrategy = fmt.Errorf("invalid strategy")
	// ErrNoServicesSpecified is returned when the number of services is zero.
	errNoServicesSpecified = fmt.Errorf("no services specified")
)

// LoadBalancerInfo is the information needed to create a load balancer.
type LoadBalancerInfo struct {
	Strategy string `yaml:"strategy"`

	// Persistent is a flag that determines if the load balancer should persist the same
	// service for the same client.
	Persistent bool `yaml:"persistent"`

	ServiceNames []string `yaml:"serviceNames"`
}

// LoadBalancer is a service that load balances between other services.
type LoadBalancer struct {
	data Data

	strategy   selection.Strategy
	services   []Service
	cookieName string

	info LoadBalancerInfo
}

func generateCookieName(host string) string {
	hash := sha256.New()
	hash.Write([]byte(fmt.Sprintf("%s%s", lbCookiePrefix, host)))

	return fmt.Sprintf("%x", hash.Sum(nil)[:lbCookieNameLength])
}

// NewLoadBalancer creates a new load balancer service.
func NewLoadBalancer(data Data, services []Service, info LoadBalancerInfo) (*LoadBalancer, error) {
	if len(services) == 0 {
		return nil, errNoServicesSpecified
	}
	s := selection.NewStrategy(info.Strategy, uint(len(services)))
	if s == nil {
		return nil, errInvalidStrategy
	}
	return &LoadBalancer{
		data:       data,
		strategy:   s,
		services:   services,
		cookieName: generateCookieName(data.Host),
		info:       info,
	}, nil
}

// Data returns the data of the load balancer service.
func (l *LoadBalancer) Data() Data {
	return l.data
}

func (l *LoadBalancer) nextServer() uint {
	next := l.strategy.Select()
	poolSize := len(l.services)
	for poolSize > 1 && l.services[next].Data().Health != nil && !l.services[next].Data().Health.Up() {
		l.services[next], l.services[poolSize-1] = l.services[poolSize-1], l.services[next]
		poolSize--
		strategy := selection.NewStrategy(l.info.Strategy, uint(poolSize))
		next = strategy.Select()
	}
	return next
}

func (l *LoadBalancer) persistentService(w http.ResponseWriter, r *http.Request) (*url.URL, error) {
	if cookie, err := r.Cookie(l.cookieName); err == nil {
		cookieNext, err := strconv.ParseUint(cookie.Value, lbCookieBase, lbCookieBitSize)
		if err == nil && (l.services[cookieNext].Data().Health == nil || l.services[cookieNext].Data().Health.Up()) {
			return l.services[cookieNext].Remote(w, r)
		}
	}
	next := l.nextServer()
	cookie := &http.Cookie{
		Name:     l.cookieName,
		Value:    fmt.Sprint(next),
		HttpOnly: true,
	}
	http.SetCookie(w, cookie)
	return l.services[next].Remote(w, r)
}

// Remote returns the remote URL of the next service in the load balancer.
func (l *LoadBalancer) Remote(w http.ResponseWriter, r *http.Request) (*url.URL, error) {
	if l.info.Persistent {
		return l.persistentService(w, r)
	}
	next := l.nextServer()
	return l.services[next].Remote(w, r)
}
