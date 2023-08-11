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
)

type serviceMap map[string]struct {
	services.Config `yaml:",inline"`

	Container *services.ContainerInfo `yaml:"container"`
	Redirect  string                  `yaml:"redirect"`
}

// Config represents a listing of services to proxy.
type Config struct {
	Services    serviceMap     `yaml:"services"`
	Log         logging.Config `yaml:"log"`
	ReadTimeout time.Duration  `yaml:"readTimeout"`
}

func (s *serviceMap) validate() error {
	for name, service := range *s {
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

	if err := config.Services.validate(); err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidConfig, err)
	}

	return &config, nil
}

// ServiceList returns a list of services from the config.
func (c *Config) ServiceList(docker dockerapi.Adapter) (services.List, error) {
	var s services.List
	for _, service := range c.Services {
		if service.Container != nil {
			container := services.NewContainer(service.Config, docker, *service.Container)
			s = append(s, container)
		} else if service.Redirect != "" {
			s = append(s, services.NewRedirect(service.Config, service.Redirect))
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
