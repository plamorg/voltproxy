package services

import "net/url"

// Redirect is a service that redirects to a remote URL.
type Redirect struct {
	host   string
	remote string
}

// NewRedirect creates a new service that redirects to a remote URL.
func NewRedirect(host string, remote string) *Redirect {
	return &Redirect{host, remote}
}

// Host returns the host name of the redirect service.
func (r Redirect) Host() string {
	return r.host
}

// Remote returns the remote URL of the redirect service.
func (r Redirect) Remote() (*url.URL, error) {
	return url.Parse(r.remote)
}
