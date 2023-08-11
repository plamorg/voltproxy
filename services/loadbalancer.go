package services

import (
	"container/ring"
	"fmt"
	"net/url"
)

var (
	errInvalidLoadBalancerStrategy = fmt.Errorf("invalid load balancer strategy")
	errNoServicesSpecified         = fmt.Errorf("no services specified")
)

type loadBalancerStrategy interface {
	next() Service
}

type roundRobin[T any] struct {
	ring *ring.Ring
}

func newRoundRobin[T any](items []T) *roundRobin[T] {
	r := ring.New(len(items))
	for _, item := range items {
		r.Value = item
		r = r.Next()
	}
	return &roundRobin[T]{ring: r}
}

func (r *roundRobin[T]) next() T {
	service := r.ring.Value.(T)
	r.ring = r.ring.Next()
	return service
}

type LoadBalancerInfo struct {
	Strategy     string   `yaml:"strategy"`
	ServiceNames []string `yaml:"serviceNames"`
}

type LoadBalancer struct {
	data

	strategy loadBalancerStrategy
}

// NewLoadBalancer creates a new load balancer service.
func NewLoadBalancer(config Config, services []Service, info LoadBalancerInfo) (*LoadBalancer, error) {
	if len(services) == 0 {
		return nil, fmt.Errorf("%s: %w", config.Host, errNoServicesSpecified)
	}
	var s loadBalancerStrategy
	switch info.Strategy {
	case "roundRobin", "":
		s = newRoundRobin(services)
	default:
		return nil, fmt.Errorf("%s: %w %s", config.Host, errInvalidLoadBalancerStrategy, info.Strategy)
	}
	return &LoadBalancer{
		data:     config.data(),
		strategy: s,
	}, nil
}

func (l *LoadBalancer) Data() data {
	return l.data
}

func (l *LoadBalancer) Remote() (*url.URL, error) {
	return l.strategy.next().Remote()
}
