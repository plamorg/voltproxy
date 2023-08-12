package services

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

const (
	lbCookiePrefix     = "voltproxy-lb-"
	lbCookieNameLength = 8
	lbCookieBase       = 10
	lbCookieBitSize    = 64
)

var (
	errInvalidLoadBalancerStrategy = fmt.Errorf("invalid load balancer strategy")
	errNoServicesSpecified         = fmt.Errorf("no services specified")
)

type loadBalancerStrategy interface {
	next() uint
}

type roundRobin struct {
	max     uint
	current uint
}

func newRoundRobin(max uint) *roundRobin {
	return &roundRobin{
		max:     max,
		current: 0,
	}
}

func (r *roundRobin) next() uint {
	current := r.current
	r.current = (r.current + 1) % r.max
	return current
}

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

	strategy   loadBalancerStrategy
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
		return nil, fmt.Errorf("%s: %w", data.Host, errNoServicesSpecified)
	}
	var s loadBalancerStrategy
	switch info.Strategy {
	case "roundRobin", "":
		s = newRoundRobin(uint(len(services)))
	default:
		return nil, fmt.Errorf("%s: %w %s", data.Host, errInvalidLoadBalancerStrategy, info.Strategy)
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

func (l *LoadBalancer) persistentRemote(w http.ResponseWriter, r *http.Request) (*url.URL, error) {
	if cookie, err := r.Cookie(l.cookieName); err == nil {
		cookieNext, err := strconv.ParseUint(cookie.Value, lbCookieBase, lbCookieBitSize)
		if err == nil {
			return l.services[cookieNext].Remote(w, r)
		}
	}
	next := l.strategy.next()
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
		return l.persistentRemote(w, r)
	}
	next := l.strategy.next()
	return l.services[next].Remote(w, r)
}
