package services

import (
	"net/url"
)

// Redirect is a service that redirects to a remote URL.
type Redirect struct {
	config Config
	remote string
}

// NewRedirect creates a new Redirect service.
func NewRedirect(config Config, remote string) *Redirect {
	return &Redirect{config, remote}
}

// Config returns the configuration of the Redirect service.
func (r *Redirect) Config() Config {
	return r.config
}

// Remote returns the remote URL of the Redirect service.
func (r *Redirect) Remote() (*url.URL, error) {
	return url.Parse(r.remote)
}
