package services

import (
	"net/url"

	"github.com/plamorg/voltproxy/middlewares"
)

// Redirect is a service that redirects to a remote URL.
type Redirect struct {
	host        string
	middlewares []middlewares.Middleware
	remote      string
}

// NewRedirect creates a new Redirect service.
func NewRedirect(host string, middlewares []middlewares.Middleware, remote string) *Redirect {
	return &Redirect{host, middlewares, remote}
}

// Host returns the host name of the Redirect service.
func (r *Redirect) Host() string {
	return r.host
}

// Remote returns the remote URL of the Redirect service.
func (r *Redirect) Remote() (*url.URL, error) {
	return url.Parse(r.remote)
}

// Middlewares returns the middlewares of the Redirect service.
func (r *Redirect) Middlewares() []middlewares.Middleware {
	return r.middlewares
}
