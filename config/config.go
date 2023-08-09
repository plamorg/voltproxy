// Package config provides a way to parse a YAML configuration file into a list of services.
package config

import (
	"bytes"
	"fmt"
	"reflect"

	"gopkg.in/yaml.v3"

	"github.com/plamorg/voltproxy/dockerapi"
	"github.com/plamorg/voltproxy/middlewares"
	"github.com/plamorg/voltproxy/services"
)

var errInvalidConfig = fmt.Errorf("invalid config")
var errMustHaveOneService = fmt.Errorf("must have exactly one service")

type middlewareData struct {
	IPAllow     *middlewares.IPAllow     `yaml:"ipAllow"`
	AuthForward *middlewares.AuthForward `yaml:"authForward"`
}

type serviceMap map[string]struct {
	Host        string          `yaml:"host"`
	TLS         bool            `yaml:"tls"`
	Middlewares *middlewareData `yaml:"middlewares"`

	Container *services.ContainerInfo `yaml:"container"`
	Redirect  string                  `yaml:"redirect"`
}

// Config represents a listing of services to proxy.
type Config struct {
	Services serviceMap `yaml:"services"`
}

func validateServices(services serviceMap) error {
	for name, service := range services {
		var (
			hasContainer = service.Container != nil
			hasAddress   = service.Redirect != ""
		)
		if hasContainer == hasAddress {
			return fmt.Errorf("%s: %w", name, errMustHaveOneService)
		}
	}
	return nil
}

// Parse parses data as YAML to return a Config.
func Parse(data []byte) (*Config, error) {
	var config Config
	decoder := yaml.NewDecoder(bytes.NewBuffer(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidConfig, err)
	}

	if err := validateServices(config.Services); err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidConfig, err)
	}

	return &config, nil
}

// ServiceList returns a list of services from the config.
func (c *Config) ServiceList(docker dockerapi.Adapter) (services.List, error) {
	var s services.List
	for _, service := range c.Services {
		var middlewareList []middlewares.Middleware
		if service.Middlewares != nil {
			middlewareList = service.Middlewares.List()
		}
		config := services.Config{
			Host:        service.Host,
			TLS:         service.TLS,
			Middlewares: middlewareList,
		}
		if service.Container != nil {
			container := services.NewContainer(docker, config, *service.Container)
			s = append(s, container)
		} else if service.Redirect != "" {
			s = append(s, services.NewRedirect(config, service.Redirect))
		}
	}
	return s, nil
}

// TLSHosts returns a list of hosts that require TLS.
func (c *Config) TLSHosts() []string {
	var hosts []string
	for _, service := range c.Services {
		if service.TLS {
			hosts = append(hosts, service.Host)
		}
	}
	return hosts
}

// List returns a list of active middlewares from the middlewareData.
func (d *middlewareData) List() []middlewares.Middleware {
	var m []middlewares.Middleware
	v := reflect.ValueOf(*d)
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsNil() {
			continue
		}
		m = append(m, v.Field(i).Interface().(middlewares.Middleware))
	}
	return m
}
