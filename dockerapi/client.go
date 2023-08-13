package dockerapi

import (
	"context"
	"log/slog"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Client is a wrapper around the Docker client.
type Client struct {
	client *client.Client
}

// NewClient returns a new Client.
func NewClient() (*Client, error) {
	c, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}
	return &Client{c}, nil
}

// LogValue logs customizable properties of the Docker client.
// These properties can be customized by setting environment variables.
// Read: client.FromEnv.
func (c Client) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("host", c.client.DaemonHost()),
		slog.String("apiVersion", c.client.ClientVersion()))
}

// ContainerList returns the list of containers from the Docker client.
func (c *Client) ContainerList() ([]Container, error) {
	clientContainers, err := c.client.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	containers := make([]Container, len(clientContainers))
	for i, container := range clientContainers {
		names := container.Names
		networks := make(map[string]IPAddress)
		for network, endpoint := range container.NetworkSettings.Networks {
			networks[network] = IPAddress(endpoint.IPAddress)
		}

		containers[i] = Container{
			Names:    names,
			Networks: networks,
		}
	}
	return containers, nil
}

var (
	_ slog.LogValuer = (*Client)(nil)
	_ Docker         = (*Client)(nil)
)
