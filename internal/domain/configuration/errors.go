package configuration

import (
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
)

const uniqueViolationSQLState = "23505"

var ErrConflict = errors.New("conflict")

type ConflictError struct {
	Constraint string
}

func (e ConflictError) Error() string {
	return uniqueConstraintMessage(e.Constraint)
}

func (e ConflictError) Unwrap() error {
	return ErrConflict
}

func wrapWriteError(op string, err error) error {
	if mapped := mapConstraintError(err); mapped != nil {
		return mapped
	}
	return fmt.Errorf("%s: %w", op, err)
}

func mapConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != uniqueViolationSQLState {
		return nil
	}
	return ConflictError{Constraint: pgErr.ConstraintName}
}

func uniqueConstraintMessage(constraint string) string {
	switch constraint {
	case "zones_floor_id_name_key":
		return "a zone with this name already exists on this floor"
	case "tables_floor_id_label_key":
		return "a table with this label already exists on this floor"
	case "floors_location_id_slug_key":
		return "a floor with this slug already exists"
	case "organisations_slug_key":
		return "an organisation with this slug already exists"
	case "locations_organisation_id_slug_key":
		return "a location with this slug already exists"
	default:
		return "a resource with this unique value already exists"
	}
}
