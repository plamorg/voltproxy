package services

import (
	"net/http"
	"net/url"
	"slices"

	"github.com/plamorg/voltproxy/dockerapi"
)

// ContainerInfo is the information needed to find a container.
type ContainerInfo struct {
	Name    string `yaml:"name"`
	Network string `yaml:"network"`
	Port    uint16 `yaml:"port"`
}

// Container is a service that is running in a Docker container.
type Container struct {
	data Data

	docker *dockerapi.Docker
	info   ContainerInfo
}

// NewContainer creates a new service from a docker container.
func NewContainer(data Data, docker dockerapi.Docker, info ContainerInfo) *Container {
	return &Container{
		data:   data,
		docker: &docker,
		info:   info,
	}
}

// Data returns the data of the Container service.
func (c *Container) Data() *Data {
	return &c.data
}

// Remote iterates through the list of containers and returns the remote of the matching container by name.
func (c *Container) Remote(_ http.ResponseWriter, _ *http.Request) (*url.URL, error) {
	containers, err := (*c.docker).ContainerList()
	if err != nil {
		return nil, err
	}
	for _, container := range containers {
		if slices.Contains(container.Names, c.info.Name) {
			if ip, ok := container.Networks[c.info.Network]; ok {
				return url.Parse(ip.URL(c.info.Port))
			}
			return nil, errNoServiceFound
		}
	}
	return nil, errNoServiceFound
}

var _ Service = (*Container)(nil)
