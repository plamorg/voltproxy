package dockerapi

// Mock is a mock implementation of the Docker interface.
// It should be used by tests so the actual Docker API is not actually called.
type Mock struct {
	// outputs is a list of outputs to return from ContainerList.
	outputs [][]Container
}

// NewMock returns a new Mock with the given container outputs.
func NewMock(outputs ...[]Container) *Mock {
	return &Mock{outputs}
}

// ContainerList returns the next container output in the list of outputs.
func (m *Mock) ContainerList() ([]Container, error) {
	output := m.outputs[0]
	m.outputs = m.outputs[1:]
	return output, nil
}

var _ Docker = (*Mock)(nil)
