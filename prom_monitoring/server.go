package prom_monitoring

import (
	"context"
	"errors"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

func RunMetricsServer(ctx context.Context) {
	if !cfg.Enabled {
		fmt.Println("Monitoring disabled")
		return
	}
	metricsMux := http.NewServeMux()
	metricsMux.Handle(cfg.Endpoint, promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}))

	server := &http.Server{
		Addr:    cfg.HostPort,
		Handler: metricsMux,
	}
	fmt.Printf("Starting HTTP metrics server [v0.2] on %s%s\n", cfg.HostPort, cfg.Endpoint)

	go func() {
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			fmt.Printf("Metrics server error: %v\n", err)
		}
	}()

	<-ctx.Done()

	_ = server.Shutdown(ctx)
	fmt.Printf("HTTP metrics server exited\n")
}
