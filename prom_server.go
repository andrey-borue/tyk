package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

var endpoint = getEnv("MONITORING_ENDPOINT", "/metrics")
var hostPort = getEnv("MONITORING_HOST_PORT", ":8005")

func init() {
	go runMetricsServer()
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func runMetricsServer() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	defer cancel()

	metricsMux := http.NewServeMux()
	metricsMux.Handle(endpoint, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:    hostPort,
		Handler: metricsMux,
	}
	fmt.Printf("Starting HTTP metrics server on %s%s\n", hostPort, endpoint)

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("Metrics server error: %v\n", err)
		}
	}()

	<-ctx.Done()

	_ = server.Shutdown(ctx)
	fmt.Printf("HTTP metrics server exited\n")
}
