// Package dockerapi provides a way to interact with the Docker API using an adapter pattern.
package dockerapi

import (
	"fmt"
	"net"
)

// IPAddress represents a Docker container's IP in a particular network.
type IPAddress string

// URL converts the IPAddress to a URL along with a port.
// Assumes the protocol is HTTP.
func (ip IPAddress) URL(port uint16) string {
	return fmt.Sprintf("http://%s", net.JoinHostPort(string(ip), fmt.Sprint(port)))
}

// Container represents a Docker container.
type Container struct {
	Names    []string
	Networks map[string]IPAddress
}

// Docker is an interface for interacting with the Docker API.
type Docker interface {
	ContainerList() ([]Container, error)
}
