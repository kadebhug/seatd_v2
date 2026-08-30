package operations

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound                = errors.New("not found")
	ErrVersionConflict         = errors.New("version conflict")
	ErrAlreadyOccupied         = errors.New("table already occupied")
	ErrAlreadyAvailable        = errors.New("table already available")
	ErrNoActiveSession         = errors.New("no active session")
	ErrInvalidAssistTransition = errors.New("invalid assist transition")
	ErrAssistAlreadyResolved   = errors.New("assist already resolved")
	ErrIdempotencyConflict     = errors.New("idempotency conflict")
	ErrActionNotEnabled        = errors.New("guest action not enabled")
	ErrRateLimited             = errors.New("rate limited")
)

type VersionConflictError struct {
	Entity   string
	ID       string
	Expected int32
	Current  int32
}

func (e VersionConflictError) Error() string {
	return fmt.Sprintf("%s %s version conflict: expected %d, current %d", e.Entity, e.ID, e.Expected, e.Current)
}

func (e VersionConflictError) Is(target error) bool {
	return target == ErrVersionConflict
}
