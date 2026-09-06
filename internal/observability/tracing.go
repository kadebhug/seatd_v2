package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/kadebhug/seatd_v2/internal/app"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.23.1"
	"go.opentelemetry.io/otel/trace"
)

const tracerName = "github.com/kadebhug/seatd_v2"

type TracingConfig struct {
	Exporter         string
	SampleRatio      float64
	DBStatements     bool
	ShutdownDisabled bool
}

func LoadTracingConfig(cfg app.Config) (TracingConfig, error) {
	exporter := strings.ToLower(strings.TrimSpace(os.Getenv("OTEL_TRACES_EXPORTER")))
	if exporter == "" {
		switch cfg.Environment {
		case app.EnvLocal, app.EnvTest:
			exporter = "console"
		default:
			exporter = "none"
			if endpointConfigured() {
				exporter = "otlp"
			}
		}
	}
	if exporter != "none" && exporter != "console" && exporter != "otlp" {
		return TracingConfig{}, fmt.Errorf("OTEL_TRACES_EXPORTER must be console, otlp, or none: %q", exporter)
	}

	sampleRatio, err := tracingSampleRatio(cfg)
	if err != nil {
		return TracingConfig{}, err
	}
	dbStatements, err := boolEnv("SEATD_TRACING_DB_STATEMENTS", false)
	if err != nil {
		return TracingConfig{}, err
	}
	return TracingConfig{
		Exporter:     exporter,
		SampleRatio:  sampleRatio,
		DBStatements: dbStatements,
	}, nil
}

func InitTracing(ctx context.Context, cfg app.Config, logger *slog.Logger) (func(context.Context) error, error) {
	tracing, err := LoadTracingConfig(cfg)
	if err != nil {
		return nil, err
	}

	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	exporter, err := spanExporter(ctx, tracing.Exporter)
	if err != nil {
		return nil, err
	}
	if exporter == nil {
		otel.SetTracerProvider(trace.NewNoopTracerProvider())
		return func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithTelemetrySDK(),
		resource.WithAttributes(
			semconv.ServiceName(cfg.Name),
			semconv.ServiceVersion(cfg.Version.Version),
			attribute.String("deployment.environment", cfg.Environment),
			attribute.String("service.commit", cfg.Version.Commit),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating trace resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(tracing.SampleRatio))),
	)
	otel.SetTracerProvider(tp)
	if logger != nil {
		logger.InfoContext(ctx, "tracing initialized", "exporter", tracing.Exporter, "sample_ratio", tracing.SampleRatio)
	}
	return tp.Shutdown, nil
}

func Tracer(scope string) trace.Tracer {
	if scope == "" {
		scope = tracerName
	}
	return otel.Tracer(scope)
}

func StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	ctx, span := Tracer(tracerName).Start(ctx, name)
	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}
	return ctx, span
}

func RecordSpanError(span trace.Span, err error) {
	if span == nil || err == nil || errors.Is(err, context.Canceled) {
		return
	}
	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func TraceLogAttrs(ctx context.Context) []any {
	sc := trace.SpanContextFromContext(ctx)
	if !sc.IsValid() {
		return nil
	}
	return []any{
		"trace_id", sc.TraceID().String(),
		"span_id", sc.SpanID().String(),
	}
}

func ExtractTraceContext(ctx context.Context, carrier propagation.TextMapCarrier) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, carrier)
}

func InjectTraceContext(ctx context.Context, carrier propagation.TextMapCarrier) {
	otel.GetTextMapPropagator().Inject(ctx, carrier)
}

func spanExporter(ctx context.Context, exporter string) (sdktrace.SpanExporter, error) {
	switch exporter {
	case "none":
		return nil, nil
	case "console":
		return stdouttrace.New(stdouttrace.WithPrettyPrint())
	case "otlp":
		return otlptracehttp.New(ctx)
	default:
		return nil, fmt.Errorf("unsupported trace exporter %q", exporter)
	}
}

func endpointConfigured() bool {
	for _, key := range []string{"OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "OTEL_EXPORTER_OTLP_ENDPOINT"} {
		if strings.TrimSpace(os.Getenv(key)) != "" {
			return true
		}
	}
	return false
}

func tracingSampleRatio(cfg app.Config) (float64, error) {
	fallback := 0.10
	if cfg.Environment == app.EnvLocal || cfg.Environment == app.EnvTest {
		fallback = 1.0
	}
	value := strings.TrimSpace(os.Getenv("SEATD_TRACING_SAMPLE_RATIO"))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil || parsed < 0 || parsed > 1 {
		return 0, fmt.Errorf("SEATD_TRACING_SAMPLE_RATIO must be between 0 and 1: %q", value)
	}
	return parsed, nil
}

func boolEnv(key string, fallback bool) (bool, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	switch strings.ToLower(value) {
	case "1", "true", "t", "yes", "y", "on":
		return true, nil
	case "0", "false", "f", "no", "n", "off":
		return false, nil
	default:
		return false, fmt.Errorf("%s must be a boolean: %q", key, value)
	}
}
