// Package config provides a way to parse a YAML configuration file into a list of services.
package config

import (
	"bytes"
	"fmt"
	"net/url"
	"reflect"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/plamorg/voltproxy/dockerapi"
	"github.com/plamorg/voltproxy/logging"
	"github.com/plamorg/voltproxy/middlewares"
	"github.com/plamorg/voltproxy/services"
	"github.com/plamorg/voltproxy/services/health"
)

var (
	errInvalidConfig      = fmt.Errorf("invalid config")
	errMustHaveOneService = fmt.Errorf("must have exactly one service")
	errNoServiceWithName  = fmt.Errorf("no service with name")
	errDuplicateHost      = fmt.Errorf("duplicate host")
)

type containerInfo struct {
	Name    string `yaml:"name"`
	Network string `yaml:"network"`
	Port    uint16 `yaml:"port"`
}

type loadBalancerInfo struct {
	ServiceNames []string `yaml:"serviceNames"`
	Strategy     string   `yaml:"strategy"`
	Persistent   bool     `yaml:"persistent"`
}

type routers struct {
	Container *containerInfo `yaml:"container"`

	Redirect string `yaml:"redirect"`

	LoadBalancer *loadBalancerInfo `yaml:"loadBalancer"`
}

func (r *routers) validate() bool {
	v := reflect.ValueOf(*r)
	count := 0
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsZero() {
			continue
		}
		if count > 0 {
			return false
		}
		count++
	}
	return count == 1
}

type serviceMap map[string]struct {
	Host        string                   `yaml:"host"`
	TLS         bool                     `yaml:"tls"`
	Middlewares *middlewares.Middlewares `yaml:"middlewares"`
	Health      *health.Info             `yaml:"health"`

	routers `yaml:",inline"`
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
		if !service.routers.validate() {
			return fmt.Errorf("%s: %w", name, errMustHaveOneService)
		}
		// Allow empty host for services. This is useful for services that
		// should only be accessed via another load balancer service.
		if service.Host != "" {
			if _, ok := hosts[service.Host]; ok {
				return fmt.Errorf("%w %s", errDuplicateHost, service.Host)
			}
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

// ServiceMap returns a mapping from hosts to services.
func (c *Config) ServiceMap(docker dockerapi.Docker) (map[string]services.Service, error) {
	nameService := make(map[string]services.Service)
	for name, service := range c.Services {
		if service.LoadBalancer != nil {
			continue
		}
		var router services.Router
		if service.Container != nil {
			router = services.NewContainer(
				service.Container.Name,
				service.Container.Network,
				service.Container.Port,
				docker,
			)
		} else if service.Redirect != "" {
			remote, err := url.Parse(service.Redirect)
			if err != nil {
				return nil, fmt.Errorf("%w: %s: %w", errInvalidConfig, name, err)
			}
			router = services.NewRedirect(*remote)
		}
		var checker health.Checker
		if service.Health != nil {
			checker = health.New(*service.Health)
		} else {
			checker = health.Always(true)
		}
		nameService[name] = services.Service{
			TLS:         service.TLS,
			Middlewares: service.Middlewares.List(),
			Health:      checker,
			Router:      router,
		}
	}

	for name, service := range c.Services {
		if service.LoadBalancer != nil {
			var lbServices []services.Service

			for _, serviceName := range service.LoadBalancer.ServiceNames {
				if s, ok := nameService[serviceName]; ok {
					lbServices = append(lbServices, s)
				} else {
					return nil, fmt.Errorf("%w: %s: %w %s", errInvalidConfig, name, errNoServiceWithName, serviceName)
				}
			}

			strategy, err := services.NewStrategy(service.LoadBalancer.Strategy)
			if err != nil {
				return nil, fmt.Errorf("%w: %s: %w", errInvalidConfig, name, err)
			}

			lb := services.NewLoadBalancer(service.Host, strategy, service.LoadBalancer.Persistent, lbServices)
			var checker health.Checker
			if service.Health != nil {
				checker = health.New(*service.Health)
			} else {
				checker = health.Always(true)
			}
			nameService[name] = services.Service{
				TLS:         service.TLS,
				Middlewares: service.Middlewares.List(),
				Health:      checker,
				Router:      lb,
			}
		}
	}

	m := make(map[string]services.Service)
	for name, service := range c.Services {
		m[service.Host] = nameService[name]
	}
	return m, nil
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
