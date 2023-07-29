// Package middlewares provides an interface for middlewares and the middlewares themselves.
package middlewares

import "net/http"

// Middleware is an interface for all middlewares.
type Middleware interface {
	Handle(next http.Handler) http.Handler
}
