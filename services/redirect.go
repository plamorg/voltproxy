package services

import "net/url"

// Redirect is a service that redirects to a remote URL.
type Redirect struct {
	data Data

	remote string
}

// NewRedirect creates a new Redirect service.
func NewRedirect(data Data, remote string) *Redirect {
	return &Redirect{
		data:   data,
		remote: remote,
	}
}

// Data returns the data of the Redirect service.
func (r *Redirect) Data() Data {
	return r.data
}

// Remote returns the remote URL of the Redirect service.
func (r *Redirect) Remote() (*url.URL, error) {
	return url.Parse(r.remote)
}
