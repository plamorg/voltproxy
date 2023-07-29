// Package services provides a way to define services that can be proxied.
package services

import (
	"net/url"

	"github.com/plamorg/voltproxy/middlewares"
)

// Service is the interface that all services must implement.
type Service interface {
	Host() string
	Remote() (*url.URL, error)
	Middlewares() []middlewares.Middleware
}
