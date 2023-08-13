package services

import (
	"errors"
	"fmt"
	"testing"

	"github.com/plamorg/voltproxy/dockerapi"
)

func TestContainerRemoteSuccess(t *testing.T) {
	dockerMock := dockerapi.NewMock([]dockerapi.Container{
		{
			Names: []string{"another", "test"},
			Networks: map[string]dockerapi.IPAddress{
				"net": "127.0.0.1",
			},
		},
	})

	container := NewContainer(Data{Host: "host"}, dockerMock, ContainerInfo{
		Name:    "test",
		Network: "net",
		Port:    1234,
	})

	expectedRemote := "http://127.0.0.1:1234"

	remote, err := container.Remote(nil, nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if remote == nil {
		t.Fatalf("expected non-nil remote")
	}

	if remote.String() != expectedRemote {
		t.Errorf("expected %s, got %s", expectedRemote, remote.String())
	}
}

func TestContainerRemoteNotInNetwork(t *testing.T) {
	dockerMock := dockerapi.NewMock([]dockerapi.Container{
		{
			Names: []string{"test"},
			Networks: map[string]dockerapi.IPAddress{
				"another": "172.24.0.3",
				"foo":     "bar",
			},
		},
	})

	container := NewContainer(Data{Host: "host"}, dockerMock, ContainerInfo{
		Name:    "test",
		Network: "net",
		Port:    25565,
	})

	_, err := container.Remote(nil, nil)

	if !errors.Is(err, errNoServiceFound) {
		t.Errorf("expected error %v, got %v", errNoServiceFound, err)
	}
}

func TestContainerRemoteNoMatchingContainer(t *testing.T) {
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

	container := NewContainer(Data{Host: "host"}, dockerMock, ContainerInfo{
		Name:    "test",
		Network: "net",
		Port:    4321,
	})

	_, err := container.Remote(nil, nil)

	if !errors.Is(err, errNoServiceFound) {
		t.Errorf("expected error %v, got %v", errNoServiceFound, err)
	}
}

var errBadDocker = fmt.Errorf("bad Docker")

type badDocker struct{}

func (badDocker) ContainerList() ([]dockerapi.Container, error) {
	return nil, errBadDocker
}

func TestContainerRemoteBadAdapter(t *testing.T) {
	container := NewContainer(Data{Host: "host"}, badDocker{}, ContainerInfo{
		Name:    "test",
		Network: "net",
		Port:    4321,
	})

	_, err := container.Remote(nil, nil)

	if !errors.Is(err, errBadDocker) {
		t.Errorf("expected error %v, got %v", errBadDocker, err)
	}
}
