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
	for name, service := range c.ServiceConfig {
		if err := service.ensureOneRouter(); err != nil {
			return nil, fmt.Errorf("%w: %w", errInvalidConfig, err)
		}
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

		nameService[name] = &services.Service{
			TLS:         service.TLS,
			Middlewares: service.Middlewares.List(),
			Health:      createHealthChecker(service.Health),
			Router:      router,
		}
	}

	if err := parseLoadBalancers(c.ServiceConfig, nameService); err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidConfig, err)
	}

	m := make(map[string]*services.Service)
	for name, service := range c.ServiceConfig {
		m[service.Host] = nameService[name]
	}
	return m, nil
}

func parseLoadBalancers(conf serviceConfig, nameService map[string]*services.Service) error {
	tempNameService := make(map[string]*services.Service)
	for name, service := range conf {
		if service.LoadBalancer == nil {
			continue
		}

		strategy, err := services.NewStrategy(service.LoadBalancer.Strategy)
		if err != nil {
			return err
		}

		var lbServices []*services.Service
		for _, serviceName := range conf[name].LoadBalancer.ServiceNames {
			if s, ok := nameService[serviceName]; ok {
				lbServices = append(lbServices, s)
			} else {
				return fmt.Errorf("%w: %s", errNoServiceWithName, serviceName)
			}
		}

		lb := services.NewLoadBalancer(
			service.Host,
			strategy,
			service.LoadBalancer.Persistent,
			lbServices,
		)

		tempNameService[name] = &services.Service{
			TLS:         service.TLS,
			Middlewares: service.Middlewares.List(),
			Health:      createHealthChecker(service.Health),
			Router:      lb,
		}
	}

	for name, service := range tempNameService {
		nameService[name] = service
	}
	return nil
}
