package services

import (
	"fmt"
	"net/url"

	"github.com/docker/docker/api/types"
)

// Container is a service that is running in a Docker container.
type Container struct {
	host      string
	ipAddress string
	port      uint16
}

// NewContainer creates a new service from a docker container,
// requires context of the docker containers currently running.
func NewContainer(containers []types.Container, host string, name string, network string, port uint16) (*Container, error) {
	for _, c := range containers {
		for _, n := range c.Names {
			if n == name {
				if endpoint, ok := c.NetworkSettings.Networks[network]; ok {
					return &Container{host, endpoint.IPAddress, port}, nil
				}
				return nil, fmt.Errorf("container %s did not have network %s", name, network)
			}
		}
	}
	return nil, fmt.Errorf("no matching container with name %s", name)
}

// Host returns the host name of the docker service.
func (c *Container) Host() string {
	return c.host
}

// Remote returns the remote URL of the docker service.
func (c *Container) Remote() (*url.URL, error) {
	return url.Parse(fmt.Sprintf("http://%s:%d", c.ipAddress, c.port))
}
