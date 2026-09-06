package observability

import (
	"context"
	"testing"

	"github.com/kadebhug/seatd_v2/internal/app"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestLoadTracingConfigDefaults(t *testing.T) {
	t.Run("local uses console exporter", func(t *testing.T) {
		t.Setenv("OTEL_TRACES_EXPORTER", "")
		cfg, err := LoadTracingConfig(testTracingConfig(app.EnvLocal))
		if err != nil {
			t.Fatalf("LoadTracingConfig() error = %v", err)
		}
		if cfg.Exporter != "console" {
			t.Fatalf("Exporter = %q, want console", cfg.Exporter)
		}
		if cfg.SampleRatio != 1 {
			t.Fatalf("SampleRatio = %v, want 1", cfg.SampleRatio)
		}
	})

	t.Run("production uses none without endpoint", func(t *testing.T) {
		t.Setenv("OTEL_TRACES_EXPORTER", "")
		t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
		t.Setenv("OTEL_EXPORTER_OTLP_TRACES_ENDPOINT", "")
		cfg, err := LoadTracingConfig(testTracingConfig(app.EnvProduction))
		if err != nil {
			t.Fatalf("LoadTracingConfig() error = %v", err)
		}
		if cfg.Exporter != "none" {
			t.Fatalf("Exporter = %q, want none", cfg.Exporter)
		}
		if cfg.SampleRatio != 0.10 {
			t.Fatalf("SampleRatio = %v, want 0.10", cfg.SampleRatio)
		}
	})

	t.Run("production uses otlp with endpoint", func(t *testing.T) {
		t.Setenv("OTEL_TRACES_EXPORTER", "")
		t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "http://collector:4318")
		cfg, err := LoadTracingConfig(testTracingConfig(app.EnvProduction))
		if err != nil {
			t.Fatalf("LoadTracingConfig() error = %v", err)
		}
		if cfg.Exporter != "otlp" {
			t.Fatalf("Exporter = %q, want otlp", cfg.Exporter)
		}
	})
}

func TestLoadTracingConfigRejectsInvalidSampleRatio(t *testing.T) {
	t.Setenv("SEATD_TRACING_SAMPLE_RATIO", "1.5")

	_, err := LoadTracingConfig(testTracingConfig(app.EnvLocal))
	if err == nil {
		t.Fatal("LoadTracingConfig() error = nil, want error")
	}
}

func TestTraceContextInjectExtract(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	previousPropagator := otel.GetTextMapPropagator()
	defer otel.SetTracerProvider(previousProvider)
	defer otel.SetTextMapPropagator(previousPropagator)

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.TraceContext{})

	ctx, producer := StartSpan(context.Background(), "producer")
	carrier := propagation.MapCarrier{}
	InjectTraceContext(ctx, carrier)
	producer.End()

	extracted := ExtractTraceContext(context.Background(), carrier)
	_, consumer := StartSpan(extracted, "consumer")
	consumer.End()

	ended := recorder.Ended()
	if len(ended) != 2 {
		t.Fatalf("ended spans = %d, want 2", len(ended))
	}
	if ended[1].SpanContext().TraceID() != ended[0].SpanContext().TraceID() {
		t.Fatal("consumer span did not continue producer trace")
	}
	if ended[1].Parent().SpanID() != ended[0].SpanContext().SpanID() {
		t.Fatal("consumer span parent did not match producer span")
	}
}

func testTracingConfig(env string) app.Config {
	return app.Config{
		Name:        "api",
		Environment: env,
		Version:     app.BuildInfo{Version: "dev", Commit: "test"},
	}
}
