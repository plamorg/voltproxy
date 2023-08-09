// Package main provides a way to proxy services.
package main

import (
	"log/slog"
	"net/http"
	"os"

	"golang.org/x/crypto/acme/autocert"

	"github.com/plamorg/voltproxy/config"
	"github.com/plamorg/voltproxy/dockerapi"
)

func initializeLogger() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	slog.SetDefault(logger)
}

func main() {
	initializeLogger()

	confBytes, err := os.ReadFile("./config.yml")
	if err != nil {
		slog.Error("Error while reading configuration file", slog.Any("error", err))
		os.Exit(1)
	}

	conf, err := config.Parse(confBytes)
	if err != nil {
		slog.Error("Error while parsing configuration file", slog.Any("error", err))
		os.Exit(1)
	}

	docker, err := dockerapi.NewClient()
	if err != nil {
		slog.Error("Error while creating Docker client", slog.Any("error", err))
		os.Exit(1)
	}

	services, err := conf.ServiceList(docker)
	if err != nil {
		slog.Error("Error while creating service list", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("Created service list", slog.Int("count", len(services)))

	tlsHosts := conf.TLSHosts()
	slog.Info("Managing certificates for hosts", slog.Any("hosts", tlsHosts))
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(tlsHosts...),
		Cache:      autocert.DirCache("_certs"),
	}

	slog.Info("Listening...")

	handler := services.Handler()
	go func() {
		slog.Error("Error from HTTP server", slog.Any("error", http.ListenAndServe(":http", certManager.HTTPHandler(handler))))
		os.Exit(1)
	}()

	tlsHandler := services.TLSHandler()
	tlsServer := &http.Server{
		Addr:      ":https",
		TLSConfig: certManager.TLSConfig(),
		Handler:   tlsHandler,
	}
	slog.Error("Error from HTTPS server", slog.Any("error", tlsServer.ListenAndServeTLS("", "")))
	os.Exit(1)
}
