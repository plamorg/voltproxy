package config

import (
	"fmt"
	"net/url"

	"github.com/plamorg/voltproxy/dockerapi"
	"github.com/plamorg/voltproxy/services"
	"github.com/plamorg/voltproxy/services/health"
)

func uniqueHosts(conf serviceConfig) bool {
	hosts := make(map[string]bool)
	for _, service := range conf {
		if service.Host == "" {
			continue
		}
		if _, ok := hosts[service.Host]; ok {
			return false
		}
		hosts[service.Host] = true
	}
	return true
}

func createHealthChecker(info *health.Info) health.Checker {
	if info == nil {
		return health.Always(true)
	}
	return health.New(*info)
}

// Services parses the config and returns a mapping from hosts to services.
func (c *Config) Services(docker dockerapi.Docker) (map[string]*services.Service, error) {
	if !uniqueHosts(c.ServiceConfig) {
		return nil, fmt.Errorf("%w: %w", errInvalidConfig, errDuplicateHost)
	}

	nameService := make(map[string]*services.Service)
	loadBalancers := make(map[string]*services.LoadBalancer)

	for name, service := range c.ServiceConfig {
		if err := service.validate(); err != nil {
			return nil, fmt.Errorf("%w: %w", errInvalidConfig, err)
		}
		var router services.Router
		if service.LoadBalancer != nil {
			strategy, err := services.NewStrategy(service.LoadBalancer.Strategy)
			if err != nil {
				return nil, fmt.Errorf("%w: %s: %w", errInvalidConfig, name, err)
			}
			loadBalancers[name] = services.NewLoadBalancer(
				service.Host,
				strategy,
				service.LoadBalancer.Persistent,
			)
		} else if service.Container != nil {
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
		nameService[name] = &services.Service{
			TLS:         service.TLS,
			Middlewares: service.Middlewares.List(),
			Health:      createHealthChecker(service.Health),
			Router:      router,
		}
	}

	for name, lb := range loadBalancers {
		var services []*services.Service
		for _, serviceName := range c.ServiceConfig[name].LoadBalancer.ServiceNames {
			if s, ok := nameService[serviceName]; ok {
				services = append(services, s)
			} else {
				return nil, fmt.Errorf("%w: %s: %w %s", errInvalidConfig, name, errNoServiceWithName, serviceName)
			}
		}
		lb.SetServices(services)
		if service, ok := nameService[name]; ok {
			service.Router = lb
			nameService[name] = service
		} else {
			return nil, fmt.Errorf("%w: %s: %w", errInvalidConfig, name, errNoServiceWithName)
		}
	}

	m := make(map[string]*services.Service)
	for name, service := range c.ServiceConfig {
		m[service.Host] = nameService[name]
	}
	return m, nil
}
