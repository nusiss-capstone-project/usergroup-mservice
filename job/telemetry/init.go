package telemetry

import (
	"context"
	"os"
	"strings"

	"github.com/nusiss-capstone-project/usergroup-mservice/job/log"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

type shutdownFunc func(context.Context) error

// Init installs a local TracerProvider for log correlation (no OTLP export required).
func Init(ctx context.Context) shutdownFunc {
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{}, propagation.Baggage{}))

	res, err := resource.New(ctx, resource.WithAttributes(
		semconv.ServiceName(serviceName()),
		semconv.ServiceVersion("1.0.0"),
	))
	if err != nil {
		log.Logger.Errorw("failed to create OpenTelemetry resource, using empty resource", "error", err)
		res = resource.Empty()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	log.Logger.Infow("local OpenTelemetry tracer enabled for log correlation",
		"traces_export_enabled", false,
		"service", serviceName(),
	)
	return func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}
}

func serviceName() string {
	if v := strings.TrimSpace(os.Getenv("OTEL_SERVICE_NAME")); v != "" {
		return v
	}
	return log.ServiceName
}

// StartRootSpan starts a root span for the job run and returns an enriched context.
func StartRootSpan(ctx context.Context, name string) (context.Context, func()) {
	tracer := otel.Tracer(log.ServiceName)
	ctx, span := tracer.Start(ctx, name)
	span.SetAttributes(attribute.String("job.name", name))
	return ctx, func() { span.End() }
}
