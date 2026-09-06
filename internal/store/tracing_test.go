package store

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestSQLOperation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sql  string
		want string
	}{
		{name: "select", sql: "select * from tables", want: "SELECT"},
		{name: "leading whitespace", sql: "\n  insert into tables(id) values($1)", want: "INSERT"},
		{name: "sqlc name", sql: "-- name: GetLocation :one\nSELECT * FROM locations", want: "GetLocation"},
		{name: "empty", sql: "", want: "UNKNOWN"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := sqlOperation(tt.sql); got != tt.want {
				t.Fatalf("sqlOperation() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPGXTracerRecordsQuerySpan(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	defer otel.SetTracerProvider(previousProvider)

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	tracer := newPGXTracer(false)
	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: "select now()"})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{})

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	if ended[0].Name() != "db.query SELECT" {
		t.Fatalf("span name = %q, want db.query SELECT", ended[0].Name())
	}
}

func TestPGXTracerRecordsQueryError(t *testing.T) {
	previousProvider := otel.GetTracerProvider()
	defer otel.SetTracerProvider(previousProvider)

	recorder := tracetest.NewSpanRecorder()
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSpanProcessor(recorder),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	otel.SetTracerProvider(tp)

	tracer := newPGXTracer(false)
	ctx := tracer.TraceQueryStart(context.Background(), nil, pgx.TraceQueryStartData{SQL: "select now()"})
	tracer.TraceQueryEnd(ctx, nil, pgx.TraceQueryEndData{Err: errors.New("query failed")})

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	if ended[0].Status().Code != codes.Error {
		t.Fatalf("span status = %v, want error", ended[0].Status().Code)
	}
}
