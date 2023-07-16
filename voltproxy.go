// Reverse proxy
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"voltproxy/config"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func containerIPAddress(containers []types.Container, container string, network string) (string, error) {
	for _, c := range containers {
		for _, name := range c.Names {
			if container == name {
				if endpointSettings, ok := c.NetworkSettings.Networks[network]; ok {
					return endpointSettings.IPAddress, nil
				}
				return "", fmt.Errorf("container %s did not have network %s", container, network)
			}
		}
	}
	return "", fmt.Errorf("no matching container with name %s", container)
}

func findServiceWithHost(host string, services map[string]voltconfig.Service) (*voltconfig.Service, error) {
	for _, service := range services {
		if service.Host == host {
			return &service, nil
		}
	}
	return nil, fmt.Errorf("could not find matching service with host %s", host)
}

func remoteFromService(containers []types.Container, service voltconfig.Service) (*url.URL, error) {
	if service.Container != nil {
		// Service is a Docker container.
		address, err := containerIPAddress(containers, service.Container.Name, service.Container.Network)
		if err != nil {
			return nil, err
		}
		return url.Parse(fmt.Sprintf("http://%s:%d", address, service.Container.Port))
	}
	// Service is a regular address.
	return url.Parse(service.Address)
}

func reverseProxy(config voltconfig.Config) (http.HandlerFunc, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return nil, err
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Find a matching service with the request's host.
		service, err := findServiceWithHost(r.Host, config.Services)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		remote, err := remoteFromService(containers, *service)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Set up reverse proxy.
		r.Host = remote.Host
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.ServeHTTP(w, r)
	}), nil
}

func main() {
	configData, err := os.ReadFile("./config.yml")
	config, err := voltconfig.Parse(configData)
	if err != nil {
		log.Fatal(err)
	}

	handler, err := reverseProxy(*config)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening...")
	log.Fatal(http.ListenAndServe(":80", handler))
}
