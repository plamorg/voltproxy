// Package config provides a way to parse a YAML configuration file into a list of services.
package config

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v3"

	"github.com/plamorg/voltproxy/services"
)

var errMustHaveOneService = fmt.Errorf("must have exactly one service")

type containerInfo struct {
	Name    string
	Network string
	Port    uint16
}

type serviceList map[string]struct {
	Host      string
	TLS       bool
	Container *containerInfo
	Redirect  string
}

// Config represents a listing of services to proxy.
type Config struct {
	Services serviceList
}

func validateServices(services serviceList) error {
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

func fetchContainers() ([]types.Container, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}
	return containers, nil
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

// ListServices returns a list of services from the config.
func (c *Config) ListServices() ([]services.Service, error) {
	containers, err := fetchContainers()
	if err != nil {
		return nil, err
	}

	var s []services.Service
	for _, service := range c.Services {
		if service.Container != nil {
			container, err := services.NewContainer(containers, service.Host, service.Container.Name, service.Container.Network, service.Container.Port)
			if err != nil {
				return nil, err
			}
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
