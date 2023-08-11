// Package config provides a way to parse a YAML configuration file into a list of services.
package config

import (
	"bytes"
	"fmt"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/plamorg/voltproxy/dockerapi"
	"github.com/plamorg/voltproxy/logging"
	"github.com/plamorg/voltproxy/services"
)

var (
	errInvalidConfig      = fmt.Errorf("invalid config")
	errMustHaveOneService = fmt.Errorf("must have exactly one service")
	errNoServiceWithName  = fmt.Errorf("no service with name")
	errDuplicateHost      = fmt.Errorf("duplicate host")
)

type serviceMap map[string]struct {
	services.Config `yaml:",inline"`

	services.Services `yaml:",inline"`
}

// Config represents a listing of services to proxy.
type Config struct {
	Services    serviceMap     `yaml:"services"`
	Log         logging.Config `yaml:"log"`
	ReadTimeout time.Duration  `yaml:"readTimeout"`
}

// validate returns true if every service has exactly one service and
// there are no duplicate hosts.
func (s *serviceMap) validate() error {
	hosts := make(map[string]bool)
	for name, service := range *s {
		if !service.Services.Validate() {
			return fmt.Errorf("%s: %w", name, errMustHaveOneService)
		}

		if _, ok := hosts[service.Host]; ok {
			return fmt.Errorf("%w %s", errDuplicateHost, service.Host)
		}

		// Allow empty host for services. This is useful for services that
		// should only be accessed via another load balancer service.
		if service.Host != "" {
			hosts[service.Host] = true
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

	if err := config.Services.validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidConfig, err)
	}

	return &config, nil
}

// ServiceList returns a list of services from the config.
func (c *Config) ServiceList(docker dockerapi.Adapter) (services.List, error) {
	m := make(map[string]services.Service)
	for name, service := range c.Services {
		if service.Container != nil {
			m[name] = services.NewContainer(service.Config, docker, *service.Container)
		} else if service.Redirect != "" {
			m[name] = services.NewRedirect(service.Config, service.Redirect)
		}
	}

	for name, service := range c.Services {
		if service.LoadBalancer != nil {
			var lbServices services.List
			for _, serviceName := range service.LoadBalancer.ServiceNames {
				if s, ok := m[serviceName]; ok {
					lbServices = append(lbServices, s)
				} else {
					return nil, fmt.Errorf("%w: %s: %w %s", errInvalidConfig, name, errNoServiceWithName, serviceName)
				}
			}
			lb, err := services.NewLoadBalancer(service.Config, lbServices, *service.LoadBalancer)
			if err != nil {
				return nil, fmt.Errorf("%w: %s: %w", errInvalidConfig, name, err)
			}
			m[name] = lb
		}
	}

	var l services.List
	for _, service := range m {
		l = append(l, service)
	}
	return l, nil
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
