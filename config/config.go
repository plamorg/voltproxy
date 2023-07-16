// Package voltconfig is responsible for config parsing
package voltconfig

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

type container struct {
	Name    string
	Network string
	Port    uint16
}

// Service relates a host to either a Docker container or address
type Service struct {
	Host      string
	Container *container
	Address   string
}

// Config represents a listing of services to proxy
type Config struct {
	Services map[string]Service
}

func validateServices(services map[string]Service) error {
	for name, service := range services {
		var (
			hasContainer = service.Container != nil
			hasAddress   = service.Address != ""
		)
		if hasContainer && hasAddress {
			return fmt.Errorf("service %s has both container and address", name)
		}
		if !hasContainer && !hasAddress {
			return fmt.Errorf("service %s has neither container nor address", name)
		}
	}
	return nil
}

// Parse parses data as YAML to return a Config
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
