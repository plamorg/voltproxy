// Package main provides a way to proxy services.
package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	"github.com/plamorg/voltproxy/config"
	"github.com/plamorg/voltproxy/services"
)

func findServiceWithHost(host string, services []services.Service) (*services.Service, error) {
	for _, service := range services {
		if service.Host() == host {
			return &service, nil
		}
	}
	return nil, fmt.Errorf("no service with host %s", host)
}

func reverseProxy(services []services.Service) (http.HandlerFunc, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		service, err := findServiceWithHost(r.Host, services)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		remote, err := (*service).Remote()
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		r.Host = remote.Host
		proxy := httputil.NewSingleHostReverseProxy(remote)
		proxy.ServeHTTP(w, r)
	}), nil
}

func main() {
	configData, err := os.ReadFile("./config.yml")
	if err != nil {
		log.Fatal(err)
	}

	c, err := config.Parse(configData)
	if err != nil {
		log.Fatal(err)
	}

	services, err := c.ListServices()
	if err != nil {
		log.Fatal(err)
	}

	handler, err := reverseProxy(services)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Listening...")
	log.Fatal(http.ListenAndServe(":80", handler))
}
