// Package selection provides load balancer selection strategies.
package selection

import (
	"math/rand"
)

// Strategy defines the interface for a load balancer selection strategy.
type Strategy interface {
	// Select returns the index of the next service to use.
	Select() uint
}

// NewStrategy creates a new Strategy based on the specified strategy string.
// If an empty string is specified, the default strategy (RoundRobin) is used.
func NewStrategy(strategy string, max uint) Strategy {
	switch strategy {
	case "roundRobin", "":
		return NewRoundRobin(max)
	case "random":
		return NewRandom(max)
	default:
		return nil
	}
}

// RoundRobin is a round-robin selection strategy.
type RoundRobin struct {
	max     uint
	current uint
}

// NewRoundRobin creates a new round-robin selection strategy.
func NewRoundRobin(max uint) *RoundRobin {
	return &RoundRobin{
		max:     max,
		current: 0,
	}
}

// Select returns the index of the next service to use using a round-robin strategy.
func (r *RoundRobin) Select() uint {
	current := r.current
	r.current = (r.current + 1) % r.max
	return current
}

// Random is a random selection strategy.
type Random struct {
	max uint
}

// NewRandom creates a new random selection strategy with a default random number generator.
func NewRandom(max uint) *Random {
	return &Random{
		max: max,
	}
}

// Select returns the index of the next service to use using a random strategy.
func (r *Random) Select() uint {
	return uint(rand.Uint64()) % r.max // #nosec
}
