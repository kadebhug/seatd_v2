package outbox

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/sync/errgroup"

	"github.com/kadebhug/seatd_v2/internal/domain/events"
)

const (
	StatusPending    = "pending"
	StatusProcessing = "processing"
	StatusProcessed  = "processed"
	StatusFailed     = "failed"
)

type Record struct {
	ID             uuid.UUID
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	EventID        uuid.UUID
	Destination    string
	Topic          string
	Payload        []byte
	Attempts       int32
	CreatedAt      time.Time
}

type Stats struct {
	PendingCount     int64
	OldestPendingAge time.Duration
}

type Consumer interface {
	Name() string
	Handle(ctx context.Context, event events.Envelope) error
}

type ConsumerFunc struct {
	ConsumerName string
	Fn           func(context.Context, events.Envelope) error
}

func (c ConsumerFunc) Name() string {
	return c.ConsumerName
}

func (c ConsumerFunc) Handle(ctx context.Context, event events.Envelope) error {
	if c.Fn == nil {
		return nil
	}
	return c.Fn(ctx, event)
}

type Worker struct {
	pool         *pgxpool.Pool
	logger       *slog.Logger
	consumerMap  map[string][]Consumer
	destinations []string
	instanceID   string
	batchSize    int32
	concurrency  int
	maxAttempts  int32
	pollInterval time.Duration
	backoffBase  time.Duration
	backoffMax   time.Duration
	claimTimeout time.Duration
}

type Config struct {
	InstanceID   string
	BatchSize    int32
	Concurrency  int
	MaxAttempts  int32
	PollInterval time.Duration
	BackoffBase  time.Duration
	BackoffMax   time.Duration
	ClaimTimeout time.Duration
}

func NewWorker(pool *pgxpool.Pool, logger *slog.Logger, consumers map[string][]Consumer, cfg Config) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	if cfg.InstanceID == "" {
		cfg.InstanceID = "worker-" + uuid.NewString()
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 25
	}
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.MaxAttempts <= 0 {
		cfg.MaxAttempts = 8
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = time.Second
	}
	if cfg.BackoffBase <= 0 {
		cfg.BackoffBase = time.Second
	}
	if cfg.BackoffMax <= 0 {
		cfg.BackoffMax = time.Minute
	}
	if cfg.ClaimTimeout <= 0 {
		cfg.ClaimTimeout = 5 * time.Minute
	}
	destinations := make([]string, 0, len(consumers))
	for destination := range consumers {
		destinations = append(destinations, destination)
	}
	return &Worker{
		pool:         pool,
		logger:       logger,
		consumerMap:  consumers,
		destinations: destinations,
		instanceID:   cfg.InstanceID,
		batchSize:    cfg.BatchSize,
		concurrency:  cfg.Concurrency,
		maxAttempts:  cfg.MaxAttempts,
		pollInterval: cfg.PollInterval,
		backoffBase:  cfg.BackoffBase,
		backoffMax:   cfg.BackoffMax,
		claimTimeout: cfg.ClaimTimeout,
	}
}

func (w *Worker) Run(ctx context.Context) error {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	for {
		processed, err := w.ProcessBatch(ctx)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return err
			}
			w.logger.WarnContext(ctx, "outbox batch failed", "error", err)
		}
		if processed > 0 {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (w *Worker) ProcessBatch(ctx context.Context) (int, error) {
	records, err := w.claim(ctx)
	if err != nil {
		return 0, err
	}
	group, groupCtx := errgroup.WithContext(ctx)
	group.SetLimit(w.concurrency)
	for _, record := range records {
		record := record
		group.Go(func() error {
			return w.processRecord(groupCtx, record)
		})
	}
	if err := group.Wait(); err != nil {
		return len(records), err
	}
	return len(records), nil
}

func (w *Worker) Stats(ctx context.Context) (Stats, error) {
	var stats Stats
	var oldest *time.Time
	err := w.adminQueryRow(ctx, `
SELECT count(*), min(created_at)
FROM outbox_records
WHERE status = 'pending'
`).Scan(&stats.PendingCount, &oldest)
	if err != nil {
		return Stats{}, fmt.Errorf("querying outbox stats: %w", err)
	}
	if oldest != nil {
		stats.OldestPendingAge = time.Since(*oldest)
	}
	return stats, nil
}

func (w *Worker) claim(ctx context.Context) ([]Record, error) {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("beginning outbox claim transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		return nil, rollback(tx, ctx, fmt.Errorf("setting worker database context: %w", err))
	}
	rows, err := tx.Query(ctx, `
WITH claimed AS (
    SELECT id
    FROM outbox_records
    WHERE (
        (
            status = 'pending'
            AND available_at <= now()
        ) OR (
            status = 'processing'
            AND locked_at < now() - ($3::bigint * interval '1 millisecond')
        )
    )
    AND (
        cardinality($4::text[]) = 0
        OR destination = ANY($4::text[])
        OR topic = ANY($4::text[])
    )
    ORDER BY created_at, id
    FOR UPDATE SKIP LOCKED
    LIMIT $1
)
UPDATE outbox_records o
SET status = 'processing',
    locked_at = now(),
    locked_by = $2,
    attempts = attempts + 1,
    last_error = NULL,
    updated_at = now()
FROM claimed
WHERE o.id = claimed.id
RETURNING o.id, o.organisation_id, o.location_id, o.event_id, o.destination, o.topic, o.payload, o.attempts, o.created_at
`, w.batchSize, w.instanceID, w.claimTimeout.Milliseconds(), w.destinations)
	if err != nil {
		return nil, rollback(tx, ctx, fmt.Errorf("claiming outbox records: %w", err))
	}
	defer rows.Close()

	records := []Record{}
	for rows.Next() {
		var record Record
		if err := rows.Scan(&record.ID, &record.OrganisationID, &record.LocationID, &record.EventID, &record.Destination, &record.Topic, &record.Payload, &record.Attempts, &record.CreatedAt); err != nil {
			return nil, rollback(tx, ctx, fmt.Errorf("scanning claimed outbox record: %w", err))
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, rollback(tx, ctx, fmt.Errorf("iterating claimed outbox records: %w", err))
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("committing outbox claim transaction: %w", err)
	}
	return records, nil
}

func (w *Worker) processRecord(ctx context.Context, record Record) error {
	var event events.Envelope
	if err := json.Unmarshal(record.Payload, &event); err != nil {
		return w.markFailed(ctx, record, fmt.Errorf("decoding event payload: %w", err))
	}

	consumers := w.consumerMap[record.Destination]
	if len(consumers) == 0 {
		consumers = w.consumerMap[record.Topic]
	}
	for _, consumer := range consumers {
		if err := w.handleConsumer(ctx, record, event, consumer); err != nil {
			return w.retryOrFail(ctx, record, err)
		}
	}
	return w.markProcessed(ctx, record)
}

func (w *Worker) handleConsumer(ctx context.Context, record Record, event events.Envelope, consumer Consumer) error {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning consumer idempotency transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		return rollback(tx, ctx, fmt.Errorf("setting consumer database context: %w", err))
	}

	var inserted bool
	if err := tx.QueryRow(ctx, `
INSERT INTO consumer_event_records (consumer_name, event_id, outbox_record_id)
VALUES ($1, $2, $3)
ON CONFLICT (consumer_name, event_id) DO NOTHING
RETURNING true
`, consumer.Name(), record.EventID, record.ID).Scan(&inserted); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return rollback(tx, ctx, fmt.Errorf("recording consumer idempotency start: %w", err))
	}
	if !inserted {
		var status string
		err := tx.QueryRow(ctx, `
SELECT status
FROM consumer_event_records
WHERE consumer_name = $1 AND event_id = $2
FOR UPDATE
`, consumer.Name(), record.EventID).Scan(&status)
		if err != nil {
			return rollback(tx, ctx, fmt.Errorf("reading consumer idempotency status: %w", err))
		}
		if status == StatusProcessed {
			if err := tx.Commit(ctx); err != nil {
				return fmt.Errorf("committing consumer idempotency skip: %w", err)
			}
			return nil
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing consumer idempotency start: %w", err)
	}

	if err := consumer.Handle(ctx, event); err != nil {
		_ = w.markConsumerFailed(ctx, consumer.Name(), record.EventID, err)
		return fmt.Errorf("consumer %s handling event %s: %w", consumer.Name(), record.EventID, err)
	}
	return w.markConsumerProcessed(ctx, consumer.Name(), record.EventID)
}

func (w *Worker) markProcessed(ctx context.Context, record Record) error {
	return w.adminExec(ctx, `
UPDATE outbox_records
SET status = 'processed',
    processed_at = now(),
    locked_at = NULL,
    locked_by = NULL,
    updated_at = now()
WHERE id = $1
`, record.ID)
}

func (w *Worker) retryOrFail(ctx context.Context, record Record, cause error) error {
	if record.Attempts >= w.maxAttempts {
		return w.markFailed(ctx, record, cause)
	}
	delay := w.backoff(record.Attempts)
	err := w.adminExec(ctx, `
UPDATE outbox_records
SET status = 'pending',
    available_at = now() + ($2::bigint * interval '1 millisecond'),
    locked_at = NULL,
    locked_by = NULL,
    last_error = $3,
    updated_at = now()
WHERE id = $1
`, record.ID, delay.Milliseconds(), cause.Error())
	if err != nil {
		return fmt.Errorf("scheduling outbox retry: %w", err)
	}
	return cause
}

func (w *Worker) markFailed(ctx context.Context, record Record, cause error) error {
	err := w.adminExec(ctx, `
UPDATE outbox_records
SET status = 'failed',
    locked_at = NULL,
    locked_by = NULL,
    last_error = $2,
    updated_at = now()
WHERE id = $1
`, record.ID, cause.Error())
	if err != nil {
		return fmt.Errorf("marking outbox record failed: %w", err)
	}
	return cause
}

func (w *Worker) markConsumerProcessed(ctx context.Context, consumerName string, eventID uuid.UUID) error {
	err := w.adminExec(ctx, `
UPDATE consumer_event_records
SET status = 'processed',
    processed_at = now(),
    last_error = NULL
WHERE consumer_name = $1 AND event_id = $2
`, consumerName, eventID)
	if err != nil {
		return fmt.Errorf("marking consumer event processed: %w", err)
	}
	return nil
}

func (w *Worker) markConsumerFailed(ctx context.Context, consumerName string, eventID uuid.UUID, cause error) error {
	err := w.adminExec(ctx, `
UPDATE consumer_event_records
SET status = 'failed',
    last_error = $3
WHERE consumer_name = $1 AND event_id = $2
`, consumerName, eventID, cause.Error())
	if err != nil {
		return fmt.Errorf("marking consumer event failed: %w", err)
	}
	return nil
}

func (w *Worker) adminExec(ctx context.Context, sql string, args ...any) error {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning admin transaction: %w", err)
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		return rollback(tx, ctx, fmt.Errorf("setting admin database context: %w", err))
	}
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return rollback(tx, ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing admin transaction: %w", err)
	}
	return nil
}

func (w *Worker) adminQueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	tx, err := w.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return errorRow{err: fmt.Errorf("beginning admin query transaction: %w", err)}
	}
	if _, err := tx.Exec(ctx, "SELECT set_config('seatd.platform_admin', 'true', true)"); err != nil {
		return errorRow{err: rollback(tx, ctx, fmt.Errorf("setting admin database context: %w", err))}
	}
	row := tx.QueryRow(ctx, sql, args...)
	return commitRow{row: row, tx: tx, ctx: ctx}
}

type errorRow struct {
	err error
}

func (r errorRow) Scan(...any) error {
	return r.err
}

type commitRow struct {
	row pgx.Row
	tx  pgx.Tx
	ctx context.Context
}

func (r commitRow) Scan(dest ...any) error {
	if err := r.row.Scan(dest...); err != nil {
		return rollback(r.tx, r.ctx, err)
	}
	if err := r.tx.Commit(r.ctx); err != nil {
		return fmt.Errorf("committing admin query transaction: %w", err)
	}
	return nil
}

func (w *Worker) backoff(attempt int32) time.Duration {
	exponent := math.Max(0, float64(attempt-1))
	delay := time.Duration(float64(w.backoffBase) * math.Pow(2, exponent))
	if delay > w.backoffMax {
		return w.backoffMax
	}
	return delay
}

func rollback(tx pgx.Tx, ctx context.Context, err error) error {
	if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
		return errors.Join(err, fmt.Errorf("rolling back transaction: %w", rollbackErr))
	}
	return err
}
