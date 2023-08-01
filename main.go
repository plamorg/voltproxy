// Package main provides a way to proxy services.
package main

import (
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/acme/autocert"

	"github.com/plamorg/voltproxy/config"
	"github.com/plamorg/voltproxy/dockerapi"
	"github.com/plamorg/voltproxy/services"
)

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func reverseProxy(list services.List, tlsHosts []string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contains(tlsHosts, r.Host) {
			http.Redirect(w, r, "https://"+r.Host+r.URL.String(), http.StatusMovedPermanently)
		} else {
			list.Proxy(r, w)
		}
	})
}

func reverseProxyTLS(list services.List, tlsHosts []string) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if contains(tlsHosts, r.Host) {
			list.Proxy(r, w)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	})
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

	handler := reverseProxy(services, tlsHosts)

	tlsHandler := reverseProxyTLS(services, tlsHosts)

	tlsServer := &http.Server{
		Addr:      ":https",
		TLSConfig: certManager.TLSConfig(),
		Handler:   tlsHandler,
	}

	log.Printf("Listening...")
	go http.ListenAndServe(":http", certManager.HTTPHandler(handler))
	tlsServer.ListenAndServeTLS("", "")
}
