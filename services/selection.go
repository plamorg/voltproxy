package services

import (
	"fmt"
	"math/rand"
	"net/http"
)

var errInvalidStrategy = fmt.Errorf("invalid strategy")

// Strategy defines the interface for a load balancer selection strategy.
type Strategy interface {
	// Select returns the index of the next service to use.
	Select([]Service, *http.Request) int
}

// NewStrategy converts a string to a Strategy.
// If the string is empty, the default strategy RoundRobin is used.
func NewStrategy(strategy string) (Strategy, error) {
	switch strategy {
	case "failover":
		return &Failover{}, nil
	case "roundRobin", "":
		return &RoundRobin{next: 0}, nil
	case "random":
		return &Random{rng: rand.Intn}, nil
	default:
		return nil, errInvalidStrategy
	}
}

// Failover is a failover selection strategy.
type Failover struct{}

// Select returns the index of the first service that is healthy.
func (f *Failover) Select(services []Service, _ *http.Request) int {
	for i, item := range services {
		if item.Health.Up() {
			return i
		}
	}
	return 0
}

// RoundRobin is a round-robin selection strategy.
type RoundRobin struct {
	next int
}

// Select returns the index of the next service to use using a round-robin strategy.
func (r *RoundRobin) Select(services []Service, _ *http.Request) int {
	for i := r.next; i < len(services)+r.next; i++ {
		if services[i%len(services)].Health.Up() {
			r.next = (i + 1) % len(services)
			return i % len(services)
		}
	}
	return 0
}

// Random is a random selection strategy.
type Random struct {
	rng func(int) int
}

// Select returns the index of the next service to use using a random strategy.
func (r *Random) Select(services []Service, _ *http.Request) int {
	var validIndices []int
	for i, item := range services {
		if item.Health.Up() {
			validIndices = append(validIndices, i)
		}
	}
	if len(validIndices) == 0 {
		return 0
	}
	return validIndices[r.rng(len(validIndices))]
}
