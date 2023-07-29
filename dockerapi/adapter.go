// Package dockerapi provides a way to interact with the Docker API using adapter pattern.
package dockerapi

import (
	"github.com/docker/docker/api/types"
)

// Adapter is an interface for interacting with the Docker API.
type Adapter interface {
	ContainerList() ([]types.Container, error)
}
