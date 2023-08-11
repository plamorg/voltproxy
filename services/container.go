package services

import (
	"fmt"
	"net/url"

	"github.com/plamorg/voltproxy/dockerapi"
)

// ErrNoMatchingContainer is returned when no matching container is found.
var ErrNoMatchingContainer = fmt.Errorf("no matching container")

// ErrContainerNotInNetwork is returned when the container is not in the desired network.
var ErrContainerNotInNetwork = fmt.Errorf("not in network")

// ContainerInfo is the information needed to find a container.
type ContainerInfo struct {
	Name    string
	Network string
	Port    uint16
}

// Container is a service that is running in a Docker container.
type Container struct {
	data

	docker *dockerapi.Adapter
	info   ContainerInfo
}

// NewContainer creates a new service from a docker container.
func NewContainer(config Config, docker dockerapi.Adapter, info ContainerInfo) *Container {
	return &Container{
		data:   config.data(),
		docker: &docker,
		info:   info,
	}
}

// Data returns the data of the Container service.
func (c *Container) Data() data {
	return c.data
}

// Remote iterates through the list of containers and returns the remote of the matching container by name.
func (c *Container) Remote() (*url.URL, error) {
	containers, err := (*c.docker).ContainerList()
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		for _, n := range container.Names {
			if n == c.info.Name {
				if endpoint, ok := container.NetworkSettings.Networks[c.info.Network]; ok {
					return url.Parse(fmt.Sprintf("http://%s:%d", endpoint.IPAddress, c.info.Port))
				}
				return nil, fmt.Errorf("%s: %w %s", c.info.Name, ErrContainerNotInNetwork, c.info.Network)
			}
		}
	}
	return nil, fmt.Errorf("%w: %s", ErrNoMatchingContainer, c.info.Name)
}
