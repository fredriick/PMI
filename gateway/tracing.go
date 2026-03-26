package gateway

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	"go.opentelemetry.io/otel/trace"
)

var tracer *Tracer

type Tracer struct {
	provider *sdktrace.TracerProvider
	tracer   trace.Tracer
}

func InitTracing(serviceName string, enabled bool) (*Tracer, error) {
	if !enabled {
		return nil, nil
	}

	exporter, err := stdouttrace.New(stdouttrace.WithPrettyPrint())
	if err != nil {
		return nil, err
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)

	otel.SetTracerProvider(tp)

	t := tp.Tracer(serviceName)

	tracer = &Tracer{
		provider: tp,
		tracer:   t,
	}

	return tracer, nil
}

func (t *Tracer) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := t.tracer.Start(ctx, name)
	for _, attr := range attrs {
		span.SetAttributes(attr)
	}
	return ctx, span
}

func (t *Tracer) EndSpan(span trace.Span) {
	span.End()
}

func (t *Tracer) Shutdown() error {
	if t != nil && t.provider != nil {
		return t.provider.Shutdown(context.Background())
	}
	return nil
}
