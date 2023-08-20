package services

import (
	"net/http"
	"net/url"
)

// Redirect is a service that redirects to a remote URL.
type Redirect struct {
	remote url.URL
}

// NewRedirect creates a new Redirect service.
func NewRedirect(remote url.URL) *Redirect {
	return &Redirect{remote}
}

// Route redirects to the remote URL.
func (r *Redirect) Route(_ http.ResponseWriter, _ *http.Request) (*url.URL, error) {
	return &r.remote, nil
}
