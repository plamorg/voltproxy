// Package middlewares provides an interface for middlewares and the middlewares themselves.
package middlewares

import (
	"net/http"
	"reflect"
)

// Config is an exhaustive structure of all middlewares.
type Config struct {
	IPAllow     *IPAllow     `yaml:"ipAllow"`
	AuthForward *AuthForward `yaml:"authForward"`
}

// List returns a list of middlewares that are not nil.
func (c *Config) List() []Middleware {
	var m []Middleware
	v := reflect.ValueOf(*c)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsNil() {
			continue
		}
		m = append(m, v.Field(i).Interface().(Middleware))
	}
	return m
}

// Middleware is an interface for all middlewares.
type Middleware interface {
	Handle(next http.Handler) http.Handler
}
