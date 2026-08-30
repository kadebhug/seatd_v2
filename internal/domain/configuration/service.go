package configuration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/seatd/seatd/internal/domain/identity"
	"github.com/seatd/seatd/internal/store/db"
)

var (
	ErrForbidden       = errors.New("forbidden")
	ErrValidation      = errors.New("validation failed")
	ErrVersionConflict = errors.New("version conflict")
	ErrNotFound        = errors.New("not found")
)

type Service struct {
	pool *pgxpool.Pool
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{pool: pool}
}

type TenantActor struct {
	OrganisationID uuid.UUID
	LocationID     uuid.UUID
	ActorRef       string
}

type OrganisationSnapshot struct {
	Organisation db.Organisation
	Locations    []db.Location
}

type LayoutSnapshot struct {
	Floors []db.Floor
	Zones  []db.Zone
	Tables []db.ListTableStatesByLocationIncludingArchivedRow
}

type UpdateOrganisationParams struct {
	TenantActor
	ID     uuid.UUID
	Slug   string
	Name   string
	Status string
}

type UpdateLocationParams struct {
	TenantActor
	ID              uuid.UUID
	Name            string
	Timezone        string
	Status          string
	OperatingConfig json.RawMessage
	FeatureFlags    json.RawMessage
}

type UpsertFloorParams struct {
	TenantActor
	ID                 uuid.UUID
	Slug               string
	Name               string
	SortOrder          int32
	Canvas             json.RawMessage
	BackgroundAssetRef string
	ExpectedVersion    int32
}

type UpsertZoneParams struct {
	TenantActor
	ID              uuid.UUID
	FloorID         uuid.UUID
	Name            string
	SortOrder       int32
	ExpectedVersion int32
}

type UpsertTableParams struct {
	TenantActor
	ID              uuid.UUID
	FloorID         uuid.UUID
	ZoneID          uuid.UUID
	Label           string
	CapacityLabel   string
	Shape           string
	Geometry        json.RawMessage
	ExpectedVersion int32
}

func (s *Service) OwnerSnapshot(ctx context.Context, actor TenantActor) (OrganisationSnapshot, error) {
	if err := actor.validate(false); err != nil {
		return OrganisationSnapshot{}, err
	}

	var result OrganisationSnapshot
	err := s.inTenantTx(ctx, actor, func(q *db.Queries) error {
		allowed, err := q.ActorHasOrganisationPermission(ctx, db.ActorHasOrganisationPermissionParams{
			OrganisationID: actor.OrganisationID,
			MemberRef:      actor.ActorRef,
			PermissionName: identity.PermissionOrganisationManage,
		})
		if err != nil {
			return fmt.Errorf("checking organisation permission: %w", err)
		}
		if !allowed {
			return ErrForbidden
		}

		org, err := q.GetOrganisation(ctx, actor.OrganisationID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("getting organisation: %w", err)
		}
		locations, err := q.ListLocationsByOrganisation(ctx, actor.OrganisationID)
		if err != nil {
			return fmt.Errorf("listing locations: %w", err)
		}
		result = OrganisationSnapshot{Organisation: org, Locations: locations}
		return nil
	})
	return result, err
}

func (s *Service) LayoutSnapshot(ctx context.Context, actor TenantActor) (LayoutSnapshot, error) {
	if err := actor.validate(true); err != nil {
		return LayoutSnapshot{}, err
	}

	var result LayoutSnapshot
	err := s.inTenantTx(ctx, actor, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, actor, identity.PermissionLayoutRead); err != nil {
			return err
		}
		floors, err := q.ListAllFloorsByLocation(ctx, db.ListAllFloorsByLocationParams{
			OrganisationID: actor.OrganisationID,
			LocationID:     actor.LocationID,
		})
		if err != nil {
			return fmt.Errorf("listing floors: %w", err)
		}
		zones, err := q.ListAllZonesByLocation(ctx, db.ListAllZonesByLocationParams{
			OrganisationID: actor.OrganisationID,
			LocationID:     actor.LocationID,
		})
		if err != nil {
			return fmt.Errorf("listing zones: %w", err)
		}
		tables, err := q.ListTableStatesByLocationIncludingArchived(ctx, db.ListTableStatesByLocationIncludingArchivedParams{
			OrganisationID: actor.OrganisationID,
			LocationID:     actor.LocationID,
		})
		if err != nil {
			return fmt.Errorf("listing table states: %w", err)
		}
		result = LayoutSnapshot{Floors: floors, Zones: zones, Tables: tables}
		return nil
	})
	return result, err
}

func (s *Service) UpdateOrganisation(ctx context.Context, arg UpdateOrganisationParams) (db.Organisation, error) {
	if err := arg.TenantActor.validate(false); err != nil {
		return db.Organisation{}, err
	}
	if arg.ID != arg.OrganisationID {
		return db.Organisation{}, ErrNotFound
	}
	if strings.TrimSpace(arg.Slug) == "" || strings.TrimSpace(arg.Name) == "" || !validStatus(arg.Status) {
		return db.Organisation{}, ErrValidation
	}

	var result db.Organisation
	err := s.inTenantTx(ctx, arg.TenantActor, func(q *db.Queries) error {
		if err := requireOrganisationPermission(ctx, q, arg.TenantActor, identity.PermissionOrganisationManage); err != nil {
			return err
		}
		updated, err := q.UpdateOrganisation(ctx, db.UpdateOrganisationParams{
			ID:     arg.ID,
			Slug:   strings.TrimSpace(arg.Slug),
			Name:   strings.TrimSpace(arg.Name),
			Status: arg.Status,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("updating organisation: %w", err)
		}
		result = updated
		return nil
	})
	return result, err
}

func (s *Service) UpdateLocation(ctx context.Context, arg UpdateLocationParams) (db.Location, error) {
	if err := arg.TenantActor.validate(true); err != nil {
		return db.Location{}, err
	}
	if arg.ID != arg.LocationID {
		return db.Location{}, ErrNotFound
	}
	if strings.TrimSpace(arg.Name) == "" || strings.TrimSpace(arg.Timezone) == "" || !validStatus(arg.Status) {
		return db.Location{}, ErrValidation
	}
	operatingConfig, err := normalizeJSONObject(arg.OperatingConfig)
	if err != nil {
		return db.Location{}, err
	}
	featureFlags, err := normalizeJSONObject(arg.FeatureFlags)
	if err != nil {
		return db.Location{}, err
	}

	var result db.Location
	err = s.inTenantTx(ctx, arg.TenantActor, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, arg.TenantActor, identity.PermissionLocationManage); err != nil {
			return err
		}
		updated, err := q.UpdateLocation(ctx, db.UpdateLocationParams{
			ID:              arg.ID,
			OrganisationID:  arg.OrganisationID,
			Name:            strings.TrimSpace(arg.Name),
			Timezone:        strings.TrimSpace(arg.Timezone),
			Status:          arg.Status,
			OperatingConfig: operatingConfig,
			FeatureFlags:    featureFlags,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("updating location: %w", err)
		}
		result = updated
		return nil
	})
	return result, err
}

func (s *Service) CreateFloor(ctx context.Context, arg UpsertFloorParams) (db.Floor, error) {
	if err := validateFloor(arg); err != nil {
		return db.Floor{}, err
	}
	canvas, err := normalizeJSONObject(arg.Canvas)
	if err != nil {
		return db.Floor{}, err
	}

	var result db.Floor
	err = s.inTenantTx(ctx, arg.TenantActor, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, arg.TenantActor, identity.PermissionLayoutWrite); err != nil {
			return err
		}
		floor, err := q.CreateFloor(ctx, db.CreateFloorParams{
			OrganisationID:     arg.OrganisationID,
			LocationID:         arg.LocationID,
			Slug:               strings.TrimSpace(arg.Slug),
			Name:               strings.TrimSpace(arg.Name),
			SortOrder:          arg.SortOrder,
			Canvas:             canvas,
			BackgroundAssetRef: nullableText(arg.BackgroundAssetRef),
		})
		if err != nil {
			return fmt.Errorf("creating floor: %w", err)
		}
		result = floor
		return nil
	})
	return result, err
}

func (s *Service) UpdateFloor(ctx context.Context, arg UpsertFloorParams) (db.Floor, error) {
	if arg.ID == uuid.Nil || arg.ExpectedVersion < 1 {
		return db.Floor{}, ErrValidation
	}
	if err := validateFloor(arg); err != nil {
		return db.Floor{}, err
	}
	canvas, err := normalizeJSONObject(arg.Canvas)
	if err != nil {
		return db.Floor{}, err
	}

	var result db.Floor
	err = s.inTenantTx(ctx, arg.TenantActor, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, arg.TenantActor, identity.PermissionLayoutWrite); err != nil {
			return err
		}
		floor, err := q.UpdateFloor(ctx, db.UpdateFloorParams{
			ID:                 arg.ID,
			OrganisationID:     arg.OrganisationID,
			LocationID:         arg.LocationID,
			Slug:               strings.TrimSpace(arg.Slug),
			Name:               strings.TrimSpace(arg.Name),
			SortOrder:          arg.SortOrder,
			Canvas:             canvas,
			BackgroundAssetRef: nullableText(arg.BackgroundAssetRef),
			Version:            arg.ExpectedVersion,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrVersionConflict
			}
			return fmt.Errorf("updating floor: %w", err)
		}
		result = floor
		return nil
	})
	return result, err
}

func (s *Service) ArchiveFloor(ctx context.Context, actor TenantActor, id uuid.UUID, expectedVersion int32) (db.Floor, error) {
	return s.changeFloorActive(ctx, actor, id, expectedVersion, false)
}

func (s *Service) RestoreFloor(ctx context.Context, actor TenantActor, id uuid.UUID, expectedVersion int32) (db.Floor, error) {
	return s.changeFloorActive(ctx, actor, id, expectedVersion, true)
}

func (s *Service) CreateZone(ctx context.Context, arg UpsertZoneParams) (db.Zone, error) {
	if err := validateZone(arg); err != nil {
		return db.Zone{}, err
	}
	var result db.Zone
	err := s.inTenantTx(ctx, arg.TenantActor, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, arg.TenantActor, identity.PermissionLayoutWrite); err != nil {
			return err
		}
		zone, err := q.CreateZone(ctx, db.CreateZoneParams{
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			FloorID:        arg.FloorID,
			Name:           strings.TrimSpace(arg.Name),
			SortOrder:      arg.SortOrder,
		})
		if err != nil {
			return fmt.Errorf("creating zone: %w", err)
		}
		result = zone
		return nil
	})
	return result, err
}

func (s *Service) UpdateZone(ctx context.Context, arg UpsertZoneParams) (db.Zone, error) {
	if arg.ID == uuid.Nil || arg.ExpectedVersion < 1 {
		return db.Zone{}, ErrValidation
	}
	if err := validateZone(arg); err != nil {
		return db.Zone{}, err
	}
	var result db.Zone
	err := s.inTenantTx(ctx, arg.TenantActor, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, arg.TenantActor, identity.PermissionLayoutWrite); err != nil {
			return err
		}
		zone, err := q.UpdateZone(ctx, db.UpdateZoneParams{
			ID:             arg.ID,
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			FloorID:        arg.FloorID,
			Name:           strings.TrimSpace(arg.Name),
			SortOrder:      arg.SortOrder,
			Version:        arg.ExpectedVersion,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrVersionConflict
			}
			return fmt.Errorf("updating zone: %w", err)
		}
		result = zone
		return nil
	})
	return result, err
}

func (s *Service) ArchiveZone(ctx context.Context, actor TenantActor, id, floorID uuid.UUID, expectedVersion int32) (db.Zone, error) {
	if err := actor.validate(true); err != nil || id == uuid.Nil || floorID == uuid.Nil || expectedVersion < 1 {
		return db.Zone{}, ErrValidation
	}
	var result db.Zone
	err := s.inTenantTx(ctx, actor, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, actor, identity.PermissionLayoutWrite); err != nil {
			return err
		}
		zone, err := q.ArchiveZone(ctx, db.ArchiveZoneParams{
			ID:             id,
			OrganisationID: actor.OrganisationID,
			LocationID:     actor.LocationID,
			FloorID:        floorID,
			Version:        expectedVersion,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrVersionConflict
			}
			return fmt.Errorf("archiving zone: %w", err)
		}
		result = zone
		return nil
	})
	return result, err
}

func (s *Service) CreateTable(ctx context.Context, arg UpsertTableParams) (db.Table, error) {
	if err := validateTable(arg, false); err != nil {
		return db.Table{}, err
	}
	geometry, err := normalizeJSONObject(arg.Geometry)
	if err != nil {
		return db.Table{}, err
	}

	var result db.Table
	err = s.inTenantTx(ctx, arg.TenantActor, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, arg.TenantActor, identity.PermissionLayoutWrite); err != nil {
			return err
		}
		table, err := q.CreateTable(ctx, db.CreateTableParams{
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			FloorID:        arg.FloorID,
			ZoneID:         arg.ZoneID,
			Label:          strings.TrimSpace(arg.Label),
			CapacityLabel:  strings.TrimSpace(arg.CapacityLabel),
			Shape:          arg.Shape,
			Geometry:       geometry,
		})
		if err != nil {
			return fmt.Errorf("creating table: %w", err)
		}
		if _, err := q.CreateTableOccupancy(ctx, db.CreateTableOccupancyParams{
			TableID:        table.ID,
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
		}); err != nil {
			return fmt.Errorf("creating table occupancy: %w", err)
		}
		result = table
		return nil
	})
	return result, err
}

func (s *Service) UpdateTable(ctx context.Context, arg UpsertTableParams) (db.Table, error) {
	if err := validateTable(arg, true); err != nil {
		return db.Table{}, err
	}
	geometry, err := normalizeJSONObject(arg.Geometry)
	if err != nil {
		return db.Table{}, err
	}

	var result db.Table
	err = s.inTenantTx(ctx, arg.TenantActor, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, arg.TenantActor, identity.PermissionLayoutWrite); err != nil {
			return err
		}
		table, err := q.UpdateTable(ctx, db.UpdateTableParams{
			ID:             arg.ID,
			OrganisationID: arg.OrganisationID,
			LocationID:     arg.LocationID,
			FloorID:        arg.FloorID,
			ZoneID:         arg.ZoneID,
			Label:          strings.TrimSpace(arg.Label),
			CapacityLabel:  strings.TrimSpace(arg.CapacityLabel),
			Shape:          arg.Shape,
			Geometry:       geometry,
			Version:        arg.ExpectedVersion,
		})
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrVersionConflict
			}
			return fmt.Errorf("updating table: %w", err)
		}
		result = table
		return nil
	})
	return result, err
}

func (s *Service) ArchiveTable(ctx context.Context, actor TenantActor, id uuid.UUID, expectedVersion int32) (db.Table, error) {
	return s.changeTableActive(ctx, actor, id, expectedVersion, false)
}

func (s *Service) RestoreTable(ctx context.Context, actor TenantActor, id uuid.UUID, expectedVersion int32) (db.Table, error) {
	return s.changeTableActive(ctx, actor, id, expectedVersion, true)
}

func (s *Service) changeFloorActive(ctx context.Context, actor TenantActor, id uuid.UUID, expectedVersion int32, active bool) (db.Floor, error) {
	if err := actor.validate(true); err != nil || id == uuid.Nil || expectedVersion < 1 {
		return db.Floor{}, ErrValidation
	}
	var result db.Floor
	err := s.inTenantTx(ctx, actor, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, actor, identity.PermissionLayoutWrite); err != nil {
			return err
		}
		var floor db.Floor
		var err error
		if active {
			floor, err = q.RestoreFloor(ctx, db.RestoreFloorParams{
				ID:             id,
				OrganisationID: actor.OrganisationID,
				LocationID:     actor.LocationID,
				Version:        expectedVersion,
			})
		} else {
			floor, err = q.ArchiveFloor(ctx, db.ArchiveFloorParams{
				ID:             id,
				OrganisationID: actor.OrganisationID,
				LocationID:     actor.LocationID,
				Version:        expectedVersion,
			})
		}
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrVersionConflict
			}
			return fmt.Errorf("changing floor active state: %w", err)
		}
		result = floor
		return nil
	})
	return result, err
}

func (s *Service) changeTableActive(ctx context.Context, actor TenantActor, id uuid.UUID, expectedVersion int32, active bool) (db.Table, error) {
	if err := actor.validate(true); err != nil || id == uuid.Nil || expectedVersion < 1 {
		return db.Table{}, ErrValidation
	}
	var result db.Table
	err := s.inTenantTx(ctx, actor, func(q *db.Queries) error {
		if err := requireLocationPermission(ctx, q, actor, identity.PermissionLayoutWrite); err != nil {
			return err
		}
		var table db.Table
		var err error
		if active {
			table, err = q.RestoreTable(ctx, db.RestoreTableParams{
				ID:             id,
				OrganisationID: actor.OrganisationID,
				LocationID:     actor.LocationID,
				Version:        expectedVersion,
			})
		} else {
			table, err = q.ArchiveTable(ctx, db.ArchiveTableParams{
				ID:             id,
				OrganisationID: actor.OrganisationID,
				LocationID:     actor.LocationID,
				Version:        expectedVersion,
			})
		}
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrVersionConflict
			}
			return fmt.Errorf("changing table active state: %w", err)
		}
		result = table
		return nil
	})
	return result, err
}

func requireOrganisationPermission(ctx context.Context, q *db.Queries, actor TenantActor, permission string) error {
	allowed, err := q.ActorHasOrganisationPermission(ctx, db.ActorHasOrganisationPermissionParams{
		OrganisationID: actor.OrganisationID,
		MemberRef:      actor.ActorRef,
		PermissionName: permission,
	})
	if err != nil {
		return fmt.Errorf("checking organisation permission: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func requireLocationPermission(ctx context.Context, q *db.Queries, actor TenantActor, permission string) error {
	allowed, err := q.ActorHasLocationPermission(ctx, db.ActorHasLocationPermissionParams{
		OrganisationID: actor.OrganisationID,
		LocationID:     actor.LocationID,
		MemberRef:      actor.ActorRef,
		PermissionName: permission,
	})
	if err != nil {
		return fmt.Errorf("checking location permission: %w", err)
	}
	if !allowed {
		return ErrForbidden
	}
	return nil
}

func (s *Service) inTenantTx(ctx context.Context, actor TenantActor, fn func(*db.Queries) error) error {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	locationValue := ""
	if actor.LocationID != uuid.Nil {
		locationValue = actor.LocationID.String()
	}
	if _, err := tx.Exec(ctx, `
SELECT set_config('seatd.platform_admin', 'false', true),
       set_config('seatd.current_organisation_id', $1::text, true),
       set_config('seatd.current_location_id', $2::text, true)
`, actor.OrganisationID.String(), locationValue); err != nil {
		return rollback(tx, ctx, fmt.Errorf("setting tenant context: %w", err))
	}
	if err := fn(db.New(tx)); err != nil {
		return rollback(tx, ctx, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}
	return nil
}

func (actor TenantActor) validate(requireLocation bool) error {
	if actor.OrganisationID == uuid.Nil || strings.TrimSpace(actor.ActorRef) == "" {
		return ErrValidation
	}
	if requireLocation && actor.LocationID == uuid.Nil {
		return ErrValidation
	}
	return nil
}

func validateFloor(arg UpsertFloorParams) error {
	if err := arg.TenantActor.validate(true); err != nil {
		return err
	}
	if strings.TrimSpace(arg.Slug) == "" || strings.TrimSpace(arg.Name) == "" || arg.SortOrder < 0 {
		return ErrValidation
	}
	return nil
}

func validateZone(arg UpsertZoneParams) error {
	if err := arg.TenantActor.validate(true); err != nil {
		return err
	}
	if arg.FloorID == uuid.Nil || strings.TrimSpace(arg.Name) == "" || arg.SortOrder < 0 {
		return ErrValidation
	}
	return nil
}

func validateTable(arg UpsertTableParams, requireExisting bool) error {
	if err := arg.TenantActor.validate(true); err != nil {
		return err
	}
	if requireExisting && (arg.ID == uuid.Nil || arg.ExpectedVersion < 1) {
		return ErrValidation
	}
	if arg.FloorID == uuid.Nil || arg.ZoneID == uuid.Nil {
		return ErrValidation
	}
	if strings.TrimSpace(arg.Label) == "" || strings.TrimSpace(arg.CapacityLabel) == "" {
		return ErrValidation
	}
	switch arg.Shape {
	case "rectangle", "circle", "square", "custom":
		return nil
	default:
		return ErrValidation
	}
}

func validStatus(status string) bool {
	return status == "active" || status == "disabled"
}

func normalizeJSONObject(raw json.RawMessage) ([]byte, error) {
	if len(raw) == 0 {
		return []byte(`{}`), nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, ErrValidation
	}
	out, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encoding json object: %w", err)
	}
	return out, nil
}

func nullableText(value string) pgtype.Text {
	value = strings.TrimSpace(value)
	return pgtype.Text{String: value, Valid: value != ""}
}

func rollback(tx pgx.Tx, ctx context.Context, err error) error {
	if rollbackErr := tx.Rollback(ctx); rollbackErr != nil && !errors.Is(rollbackErr, pgx.ErrTxClosed) {
		return errors.Join(err, fmt.Errorf("rolling back transaction: %w", rollbackErr))
	}
	return err
}
