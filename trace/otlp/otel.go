package otlp

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/TykTechnologies/tyk/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.12.0"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Name is the name of this tracer.
const Name = "otlp"

type Trace struct {
	trace.Tracer
	io.Closer
}

func (Trace) Name() string {
	return Name
}

type Logger interface {
	Errorf(msg string, args ...interface{})
	Infof(msg string, args ...interface{})
}

type wrapLogger struct {
	Logger
}

func (w wrapLogger) Error(msg string) {
	w.Errorf("%s", msg)
}

func InitOtelProvider(resourceName string, opts config.OpenTelemetryConfig) (func(context.Context) error, error) {
	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			// the service name used to display traces in backends
			semconv.ServiceNameKey.String(resourceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	collectorURL := opts.URL

	connTimeout := time.Duration(opts.Timeout) * time.Second

	// If the OpenTelemetry Collector is running on a local cluster (minikube or
	// microk8s), it should be accessible through the NodePort service at the
	// `localhost:30080` endpoint. Otherwise, replace `localhost` with the
	// endpoint of your cluster. If you run the app inside k8s, then you can
	// probably connect directly to the service through dns
	ctx, cancel := context.WithTimeout(ctx, connTimeout)
	defer cancel()
	conn, err := grpc.DialContext(ctx, collectorURL, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, fmt.Errorf("failed to create gRPC connection to collector: %w", err)
	}
	// Set up a trace exporter
	traceExporter, err := otlptracegrpc.New(ctx, otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, fmt.Errorf("failed to create trace exporter: %w", err)
	}

	// Register the trace exporter with a TracerProvider, using a batch
	// span processor to aggregate spans before export.
	bsp := sdktrace.NewBatchSpanProcessor(traceExporter)
	tracerProvider := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
		sdktrace.WithSpanProcessor(bsp),
	)
	otel.SetTracerProvider(tracerProvider)

	// set global propagator to tracecontext (the default is no-op).
	otel.SetTextMapPropagator(propagation.TraceContext{})

	// Shutdown will flush any remaining spans and shut down the exporter.
	return tracerProvider.Shutdown, nil
}

func Init(service string, opts map[string]interface{}, log Logger) (*Trace, error) {
	c, err := Load(opts)
	if err != nil {
		return nil, err
	}
	if c.URL == "" {
		log.Errorf("missing OTel URL, defaulting to :4317")
		c.URL = ":4317"
	}
	if c.Timeout == 0 {
		log.Errorf("missing OTel Timeout, defaulting to 10 seconds")
		c.Timeout = 10
	}
	_, err = InitOtelProvider(service, *c)
	if err != nil {
		return nil, err
	}

	tr := otel.Tracer("tyk-gw")

	return &Trace{Tracer: tr}, nil
}

// Load retusn a zipkin configuration from the opts.
func Load(opts map[string]interface{}) (*config.OpenTelemetryConfig, error) {
	var c config.OpenTelemetryConfig
	if err := config.DecodeJSON(&c, opts); err != nil {
		return nil, err
	}
	return &c, nil
}