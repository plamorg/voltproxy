// Package main provides a way to proxy services.
package main

import (
	"log"
	"net/http"
	"os"

	"golang.org/x/crypto/acme/autocert"

	"github.com/plamorg/voltproxy/config"
	"github.com/plamorg/voltproxy/dockerapi"
)

func main() {
	confBytes, err := os.ReadFile("./config.yml")
	if err != nil {
		log.Fatal(err)
	}

	conf, err := config.Parse(confBytes)
	if err != nil {
		log.Fatal(err)
	}

	docker, err := dockerapi.NewClient()
	if err != nil {
		log.Fatal(err)
	}

	services, err := conf.ServiceList(docker)
	if err != nil {
		log.Fatal(err)
	}

	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(conf.TLSHosts()...),
		Cache:      autocert.DirCache("_certs"),
	}

	log.Printf("Listening...")

	handler := services.Handler()
	go http.ListenAndServe(":http", certManager.HTTPHandler(handler))

	tlsHandler := services.TLSHandler()
	tlsServer := &http.Server{
		Addr:      ":https",
		TLSConfig: certManager.TLSConfig(),
		Handler:   tlsHandler,
	}
	tlsServer.ListenAndServeTLS("", "")
}
