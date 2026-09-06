package store

import (
	"context"
	"strings"
	"unicode"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const dbTracerName = "github.com/kadebhug/seatd_v2/internal/store"

type pgxTracer struct {
	includeStatement bool
}

type spanContextKey struct{}

func newPGXTracer(includeStatement bool) *pgxTracer {
	return &pgxTracer{includeStatement: includeStatement}
}

func (t *pgxTracer) TraceQueryStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryStartData) context.Context {
	operation := sqlOperation(data.SQL)
	attrs := []attribute.KeyValue{attribute.String("db.system.name", "postgresql")}
	if operation != "" {
		attrs = append(attrs, attribute.String("db.operation.name", operation))
	}
	if t.includeStatement {
		attrs = append(attrs, attribute.String("db.query.text", strings.TrimSpace(data.SQL)))
	}
	ctx, span := otel.Tracer(dbTracerName).Start(ctx, "db.query "+operation, trace.WithAttributes(attrs...))
	return context.WithValue(ctx, spanContextKey{}, span)
}

func (t *pgxTracer) TraceQueryEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceQueryEndData) {
	endSpan(ctx, data.Err)
}

func (t *pgxTracer) TraceBatchStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceBatchStartData) context.Context {
	ctx, span := otel.Tracer(dbTracerName).Start(ctx, "db.batch", trace.WithAttributes(attribute.String("db.system.name", "postgresql")))
	return context.WithValue(ctx, spanContextKey{}, span)
}

func (t *pgxTracer) TraceBatchQuery(context.Context, *pgx.Conn, pgx.TraceBatchQueryData) {}

func (t *pgxTracer) TraceBatchEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceBatchEndData) {
	endSpan(ctx, data.Err)
}

func (t *pgxTracer) TraceCopyFromStart(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromStartData) context.Context {
	ctx, span := otel.Tracer(dbTracerName).Start(ctx, "db.copy", trace.WithAttributes(
		attribute.String("db.system.name", "postgresql"),
		attribute.String("db.collection.name", strings.Join(data.TableName, ".")),
	))
	return context.WithValue(ctx, spanContextKey{}, span)
}

func (t *pgxTracer) TraceCopyFromEnd(ctx context.Context, _ *pgx.Conn, data pgx.TraceCopyFromEndData) {
	endSpan(ctx, data.Err)
}

func (t *pgxTracer) TracePrepareStart(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareStartData) context.Context {
	operation := sqlOperation(data.SQL)
	ctx, span := otel.Tracer(dbTracerName).Start(ctx, "db.prepare "+operation, trace.WithAttributes(
		attribute.String("db.system.name", "postgresql"),
		attribute.String("db.operation.name", operation),
	))
	return context.WithValue(ctx, spanContextKey{}, span)
}

func (t *pgxTracer) TracePrepareEnd(ctx context.Context, _ *pgx.Conn, data pgx.TracePrepareEndData) {
	endSpan(ctx, data.Err)
}

func (t *pgxTracer) TraceConnectStart(ctx context.Context, _ pgx.TraceConnectStartData) context.Context {
	ctx, span := otel.Tracer(dbTracerName).Start(ctx, "db.connect", trace.WithAttributes(attribute.String("db.system.name", "postgresql")))
	return context.WithValue(ctx, spanContextKey{}, span)
}

func (t *pgxTracer) TraceConnectEnd(ctx context.Context, _ pgx.TraceConnectEndData) {
	endSpan(ctx, nil)
}

func (t *pgxTracer) TraceAcquireStart(ctx context.Context, _ *pgxpool.Pool, _ pgxpool.TraceAcquireStartData) context.Context {
	ctx, span := otel.Tracer(dbTracerName).Start(ctx, "db.pool.acquire", trace.WithAttributes(attribute.String("db.system.name", "postgresql")))
	return context.WithValue(ctx, spanContextKey{}, span)
}

func (t *pgxTracer) TraceAcquireEnd(ctx context.Context, _ *pgxpool.Pool, data pgxpool.TraceAcquireEndData) {
	endSpan(ctx, data.Err)
}

func (t *pgxTracer) TraceRelease(_ *pgxpool.Pool, _ pgxpool.TraceReleaseData) {}

func endSpan(ctx context.Context, err error) {
	span, ok := ctx.Value(spanContextKey{}).(trace.Span)
	if !ok {
		return
	}
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	}
	span.End()
}

func sqlOperation(sql string) string {
	sql = strings.TrimSpace(sql)
	if sql == "" {
		return "UNKNOWN"
	}
	if strings.HasPrefix(sql, "-- name:") {
		line, _, _ := strings.Cut(sql, "\n")
		fields := strings.Fields(strings.TrimPrefix(line, "-- name:"))
		if len(fields) > 0 {
			return fields[0]
		}
	}
	for _, field := range strings.Fields(sql) {
		field = strings.TrimFunc(field, func(r rune) bool {
			return !unicode.IsLetter(r)
		})
		if field != "" && !strings.HasPrefix(field, "--") {
			return strings.ToUpper(field)
		}
	}
	return "UNKNOWN"
}
