package configuration

import (
	"errors"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestMapConstraintErrorUniqueZoneName(t *testing.T) {
	t.Parallel()

	err := fmt.Errorf("creating zone: %w", &pgconn.PgError{
		Code:           uniqueViolationSQLState,
		ConstraintName: "zones_floor_id_name_key",
	})

	mapped := mapConstraintError(err)
	var conflict ConflictError
	if !errors.As(mapped, &conflict) {
		t.Fatalf("mapConstraintError() = %v, want ConflictError", mapped)
	}
	if !errors.Is(mapped, ErrConflict) {
		t.Fatalf("mapConstraintError() = %v, want ErrConflict", mapped)
	}
	if conflict.Error() != "a zone with this name already exists on this floor" {
		t.Fatalf("message = %q", conflict.Error())
	}
}

func TestMapConstraintErrorIgnoresOtherSQLStates(t *testing.T) {
	t.Parallel()

	err := &pgconn.PgError{Code: "23503", ConstraintName: "zones_floor_id_fkey"}
	if mapped := mapConstraintError(err); mapped != nil {
		t.Fatalf("mapConstraintError() = %v, want nil", mapped)
	}
}

func TestUniqueConstraintMessageFallback(t *testing.T) {
	t.Parallel()

	got := uniqueConstraintMessage("unknown_constraint")
	want := "a resource with this unique value already exists"
	if got != want {
		t.Fatalf("uniqueConstraintMessage() = %q, want %q", got, want)
	}
}
