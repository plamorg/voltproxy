package health

import (
	"net/http"
	"net/url"
)

// Always is a health checker that always returns the same value.
// It is used when no health check is specified.
type Always bool

// Launch does nothing.
func (a Always) Launch(func(w http.ResponseWriter, r *http.Request) (*url.URL, error)) {}

// Up always returns true.
func (a Always) Up() bool {
	return bool(a)
}

// Check always returns a nil channel.
// Receiving from this channel will block forever.
func (a Always) Check() <-chan Result {
	return nil
}
