// Package main provides a way to proxy services.
package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"os"

	"golang.org/x/crypto/acme/autocert"

	"github.com/plamorg/voltproxy/config"
	"github.com/plamorg/voltproxy/dockerapi"
	"github.com/plamorg/voltproxy/services"
)

var errNoServiceWithHost = fmt.Errorf("no service with host")

func findServiceWithHost(host string, services []services.Service) (services.Service, error) {
	for _, service := range services {
		if service.Host() == host {
			return service, nil
		}
	}
	return nil, fmt.Errorf("%w: %s", errNoServiceWithHost, host)
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func directToService(w http.ResponseWriter, r *http.Request, s []services.Service) {
	service, err := findServiceWithHost(r.Host, s)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}

	remote, err := service.Remote()
	if errors.Is(err, services.ErrNoMatchingContainer) {
		log.Println(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	r.Host = remote.Host
	proxy := httputil.NewSingleHostReverseProxy(remote)
	proxy.ServeHTTP(w, r)
}

func reverseProxy(services []services.Service, tlsHosts []string) (http.HandlerFunc, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contains(tlsHosts, r.Host) {
			http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
		} else {
			directToService(w, r, services)
		}
	}), nil
}

func reverseProxyTLS(services []services.Service, tlsHosts []string) (http.HandlerFunc, error) {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contains(tlsHosts, r.Host) {
			directToService(w, r, services)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}), nil
}

func main() {
	confBytes, err := os.ReadFile("./config.yml")
	if err != nil {
		log.Fatal(err)
	}

	conf, err := config.Parse(confBytes)
	if err != nil {
		log.Fatal(err)
	}

	cli, err := dockerapi.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	services, err := conf.ServiceList(cli)
	if err != nil {
		log.Fatal(err)
	}

	tlsHosts := conf.TLSHosts()

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(tlsHosts...),
		Cache:      autocert.DirCache("_certs"),
	}

	handler, err := reverseProxy(services, tlsHosts)
	if err != nil {
		log.Fatal(err)
	}

	tlsHandler, err := reverseProxyTLS(services, tlsHosts)
	if err != nil {
		log.Fatal(err)
	}

	tlsServer := &http.Server{
		Addr:      ":https",
		TLSConfig: certManager.TLSConfig(),
		Handler:   tlsHandler,
	}

	log.Printf("Listening...")
	go http.ListenAndServe(":http", certManager.HTTPHandler(handler))
	log.Fatal(tlsServer.ListenAndServeTLS("", ""))
}
