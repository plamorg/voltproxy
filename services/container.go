package services

import (
	"fmt"
	"net/http"
	"net/url"
	"slices"

	"github.com/plamorg/voltproxy/dockerapi"
)

var (
	errNoContainerFound = fmt.Errorf("no container found")
	errNoNetworkFound   = fmt.Errorf("no network found")
)

// Container is a service that is running in a Docker container.
type Container struct {
	name    string
	network string
	port    uint16

	docker *dockerapi.Docker
}

// NewContainer creates a new service from a docker container.
func NewContainer(name string, network string, port uint16, docker dockerapi.Docker) *Container {
	return &Container{
		name:    name,
		network: network,
		port:    port,
		docker:  &docker,
	}
}

// Route iterates through the list of containers and returns the remote of the matching container by name.
func (c *Container) Route(_ http.ResponseWriter, _ *http.Request) (*url.URL, error) {
	containers, err := (*c.docker).ContainerList()
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		if slices.Contains(container.Names, c.name) {
			if ip, ok := container.Networks[c.network]; ok {
				return url.Parse(ip.URL(c.port))
			}
			return nil, errNoNetworkFound
		}
	}
	return nil, errNoContainerFound
}
