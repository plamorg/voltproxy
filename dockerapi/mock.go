package dockerapi

import "github.com/docker/docker/api/types"

// Mock is a mock implementation of the Adapter interface.
type Mock struct {
	containers []types.Container
}

// NewMock returns a new Mock.
func NewMock(containers []types.Container) *Mock {
	return &Mock{containers}
}

// ContainerList returns the list of containers from the Mock's containers field.
func (m *Mock) ContainerList() ([]types.Container, error) {
	return m.containers, nil
}
