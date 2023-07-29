package dockerapi

import (
	"context"

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

// ContainerList returns the list of containers from the Docker client.
func (c *Client) ContainerList() ([]types.Container, error) {
	return c.client.ContainerList(context.Background(), types.ContainerListOptions{})
}
