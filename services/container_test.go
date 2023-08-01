package services

import (
	"errors"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/network"
	"github.com/plamorg/voltproxy/dockerapi"
)

func TestContainerRemoteSuccess(t *testing.T) {
	adapter := dockerapi.NewMock([]types.Container{
		{
			ID:    "a",
			Names: []string{"another", "test"},
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"net": {
						IPAddress: "127.0.0.1",
					},
				},
			},
		},
	})

	container := NewContainer(adapter, Config{Host: "host"}, ContainerInfo{
		Name:    "test",
		Network: "net",
		Port:    1234,
	})

	expectedRemote := "http://127.0.0.1:1234"

	remote, err := container.Remote()

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
	adapter := dockerapi.NewMock([]types.Container{
		{
			ID:    "a",
			Names: []string{"test"},
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"another": {
						IPAddress: "172.24.0.3",
					},
					"foo": {
						IPAddress: "bar",
					},
				},
			},
		},
	})

	container := NewContainer(adapter, Config{Host: "host"}, ContainerInfo{
		Name:    "test",
		Network: "net",
		Port:    25565,
	})

	_, err := container.Remote()

	if !errors.Is(err, ErrContainerNotInNetwork) {
		t.Errorf("expected error %v, got %v", ErrContainerNotInNetwork, err)
	}
}

func TestContainerRemoteNoMatchingContainer(t *testing.T) {
	adapter := dockerapi.NewMock([]types.Container{
		{
			ID:    "a",
			Names: []string{"foo", "bar", ""},
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"net": {
						IPAddress: "172.24.0.3",
					},
				},
			},
		},
		{
			ID:    "b",
			Names: []string{"baz"},
			NetworkSettings: &types.SummaryNetworkSettings{
				Networks: map[string]*network.EndpointSettings{
					"net": {
						IPAddress: "172.21.0.4",
					},
				},
			},
		},
	})

	container := NewContainer(adapter, Config{Host: "host"}, ContainerInfo{
		Name:    "test",
		Network: "net",
		Port:    4321,
	})

	_, err := container.Remote()

	if !errors.Is(err, ErrNoMatchingContainer) {
		t.Errorf("expected error %v, got %v", ErrContainerNotInNetwork, err)
	}
}
