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
	"github.com/plamorg/voltproxy/services"
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

func logPanic(msg string, err error) {
	slog.Error(msg, slog.Any("error", err))
	os.Exit(1)
}

func main() {
	confContent, err := os.ReadFile("./config.yml")
	if err != nil {
		logPanic("Error while reading configuration file", err)
	}

	conf, err := config.New(confContent)
	if err != nil {
		logPanic("Error while parsing configuration file", err)
	}

	if err = conf.LogConfig.Initialize(); err != nil {
		logPanic("Error while initializing logging", err)
	}
	slog.Info("Logging enabled", slog.Any("logger", conf.LogConfig))

	docker, err := dockerapi.NewClient()
	if err != nil {
		logPanic("Error while connecting to Docker", err)
	}
	slog.Info("Connected to Docker", slog.Any("docker", docker))

	serviceMap, err := conf.Services(docker)
	if err != nil {
		logPanic("Error while fetching services", err)
	}

	services.LaunchHealthChecks(serviceMap)

	tlsHosts := conf.TLSHosts()
	slog.Info("Managing certificates", slog.Any("hosts", tlsHosts))
	certManager := autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(tlsHosts...),
		Cache:      autocert.DirCache("_certs"),
	}

	slog.Info("Accepting connections on :80 and :443")
	go listen(certManager.HTTPHandler(services.Handler(serviceMap)), conf.ReadTimeout)
	listenTLS(services.TLSHandler(serviceMap), conf.ReadTimeout, certManager.TLSConfig())
}
