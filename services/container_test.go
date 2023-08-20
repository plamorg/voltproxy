package services

import (
	"errors"
	"fmt"
	"testing"

	"github.com/plamorg/voltproxy/dockerapi"
)

func TestContainerRouteSuccess(t *testing.T) {
	dockerMock := dockerapi.NewMock([]dockerapi.Container{
		{
			Names: []string{"another", "test"},
			Networks: map[string]dockerapi.IPAddress{
				"net": "127.0.0.1",
			},
		},
	})

	container := NewContainer("test", "net", 1234, dockerMock)

	route, err := container.Route(nil, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if route == nil {
		t.Fatalf("expected non-nil remote")
	}

	expectedRemote := "http://127.0.0.1:1234"
	if route.String() != expectedRemote {
		t.Errorf("expected %s, got %s", expectedRemote, route.String())
	}
}

// TODO: Combine these error tests into one testing function.

func TestContainerRouteNotInNetwork(t *testing.T) {
	dockerMock := dockerapi.NewMock([]dockerapi.Container{
		{
			Names: []string{"test"},
			Networks: map[string]dockerapi.IPAddress{
				"another": "172.24.0.3",
				"foo":     "bar",
			},
		},
	})

	container := NewContainer("test", "net", 25565, dockerMock)

	_, err := container.Route(nil, nil)

	if !errors.Is(err, errNoNetworkFound) {
		t.Errorf("expected error %v, got %v", errNoNetworkFound, err)
	}
}

func TestContainerRouteNoMatchingContainer(t *testing.T) {
	dockerMock := dockerapi.NewMock([]dockerapi.Container{
		{
			Names: []string{"foo", "bar", ""},
			Networks: map[string]dockerapi.IPAddress{
				"net": "172.24.0.3",
			},
		},
		{
			Names: []string{"baz"},
			Networks: map[string]dockerapi.IPAddress{
				"net": "172.21.0.4",
			},
		},
	})

	container := NewContainer("test", "net", 4321, dockerMock)

	_, err := container.Route(nil, nil)

	if !errors.Is(err, errNoContainerFound) {
		t.Errorf("expected error %v, got %v", errNoContainerFound, err)
	}
}

var errBadDocker = fmt.Errorf("bad Docker")

type badDocker struct{}

func (badDocker) ContainerList() ([]dockerapi.Container, error) {
	return nil, errBadDocker
}

func TestContainerRouteBadDocker(t *testing.T) {
	container := NewContainer("test", "net", 1234, badDocker{})

	_, err := container.Route(nil, nil)

	if !errors.Is(err, errBadDocker) {
		t.Errorf("expected error %v, got %v", errBadDocker, err)
	}
}
