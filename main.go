// Package main provides a way to proxy services.
package main

import (
	"crypto/tls"
	"log/slog"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/acme/autocert"

	"github.com/plamorg/voltproxy/config"
	"github.com/plamorg/voltproxy/dockerapi"
)

func listen(handler http.Handler, timeout time.Duration) {
	server := &http.Server{
		Addr:        ":http",
		Handler:     handler,
		ReadTimeout: timeout,
	}
	slog.Error("Error from HTTP server",
		slog.Any("error", server.ListenAndServe()),
	)
	os.Exit(1)
}

func listenTLS(handler http.Handler, timeout time.Duration, tlsConfig *tls.Config) {
	tlsServer := &http.Server{
		Addr:        ":https",
		TLSConfig:   tlsConfig,
		Handler:     handler,
		ReadTimeout: timeout,
	}
	slog.Error("Error from HTTPS server", slog.Any("error", tlsServer.ListenAndServeTLS("", "")))
	os.Exit(1)
}

func main() {
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

	if err = conf.Log.Initialize(); err != nil {
		slog.Error("Error while initializing logging", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("Initialized logging", slog.Any("logger", conf.Log))

	docker, err := dockerapi.NewClient()
	if err != nil {
		slog.Error("Error while creating Docker client", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("Created Docker client", slog.Any("client", docker))

	services, err := conf.ServiceList(docker)
	if err != nil {
		slog.Error("Error while creating service list", slog.Any("error", err))
		os.Exit(1)
	}
	slog.Info("Created service list", slog.Int("count", len(services)))

	services.LaunchHealthChecks()

	tlsHosts := conf.TLSHosts()
	slog.Info("Managing certificates for hosts", slog.Any("hosts", tlsHosts))
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(tlsHosts...),
		Cache:      autocert.DirCache("_certs"),
	}

	slog.Info("Listening...", slog.String("readTimeout", conf.ReadTimeout.String()))

	go listen(certManager.HTTPHandler(services.Handler()), conf.ReadTimeout)
	listenTLS(services.TLSHandler(), conf.ReadTimeout, certManager.TLSConfig())
}
