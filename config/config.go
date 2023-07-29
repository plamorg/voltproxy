// Package config provides a way to parse a YAML configuration file into a list of services.
package config

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/plamorg/voltproxy/dockerapi"
	"github.com/plamorg/voltproxy/services"
)

var errMustHaveOneService = fmt.Errorf("must have exactly one service")

type serviceMap map[string]struct {
	Host      string
	TLS       bool
	Container *services.ContainerInfo
	Redirect  string
}

// Config represents a listing of services to proxy.
type Config struct {
	Services serviceMap
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
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if err := validateServices(config.Services); err != nil {
		return nil, err
	}

	return &config, nil
}

// ServiceList returns a list of services from the config.
func (c *Config) ServiceList(adapter dockerapi.Adapter) ([]services.Service, error) {
	var s []services.Service
	for _, service := range c.Services {
		if service.Container != nil {
			container := services.NewContainer(adapter, service.Host, *service.Container)
			s = append(s, container)
		} else if service.Redirect != "" {
			s = append(s, services.NewRedirect(service.Host, service.Redirect))
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
