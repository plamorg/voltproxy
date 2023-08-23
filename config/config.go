// Package config provides a way to parse a YAML configuration file into a list of services.
package config

import (
	"bytes"
	"fmt"
	"reflect"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/plamorg/voltproxy/logging"
	"github.com/plamorg/voltproxy/middlewares"
	"github.com/plamorg/voltproxy/services/health"
)

var (
	errInvalidConfig     = fmt.Errorf("invalid config")
	errMustHaveOneRouter = fmt.Errorf("must have exactly one router")
	errNoServiceWithName = fmt.Errorf("no service with name")
	errDuplicateHost     = fmt.Errorf("duplicate host")
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
	Container    *containerInfo    `yaml:"container"`
	Redirect     string            `yaml:"redirect"`
	LoadBalancer *loadBalancerInfo `yaml:"loadBalancer"`
}

func (r *routers) ensureOneRouter() error {
	v := reflect.ValueOf(*r)
	count := 0
	for i := 0; i < v.NumField(); i++ {
		if v.Field(i).IsZero() {
			continue
		}
		if count > 0 {
			return errMustHaveOneRouter
		}
		count++
	}
	if count == 0 {
		return errMustHaveOneRouter
	}
	return nil
}

type serviceConfig map[string]struct {
	Host        string                   `yaml:"host"`
	TLS         bool                     `yaml:"tls"`
	Middlewares *middlewares.Middlewares `yaml:"middlewares"`
	Health      *health.Info             `yaml:"health"`

	routers `yaml:",inline"`
}

// Config represents a listing of services to proxy.
type Config struct {
	ServiceConfig serviceConfig  `yaml:"services"`
	LogConfig     logging.Config `yaml:"log"`
	ReadTimeout   time.Duration  `yaml:"readTimeout"`
}

// New parses the given YAML data into a Config.
func New(data []byte) (*Config, error) {
	decoder := yaml.NewDecoder(bytes.NewBuffer(data))
	decoder.KnownFields(true)
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("%w: %w", errInvalidConfig, err)
	}
	return &config, nil
}

// TLSHosts returns a list of hosts that require TLS.
func (c *Config) TLSHosts() []string {
	var hosts []string
	for _, service := range c.ServiceConfig {
		if service.TLS {
			hosts = append(hosts, service.Host)
		}
	}
	return hosts
}
