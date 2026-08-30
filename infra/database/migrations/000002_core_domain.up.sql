CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE IF NOT EXISTS organisations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug text NOT NULL UNIQUE,
    name text NOT NULL,
    status text NOT NULL DEFAULT 'active',
    legacy_restaurant_id text UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (slug <> ''),
    CHECK (name <> ''),
    CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE IF NOT EXISTS locations (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL REFERENCES organisations (id) ON DELETE RESTRICT,
    slug text NOT NULL,
    name text NOT NULL,
    timezone text NOT NULL DEFAULT 'UTC',
    status text NOT NULL DEFAULT 'active',
    legacy_restaurant_id text UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organisation_id, slug),
    UNIQUE (id, organisation_id),
    CHECK (slug <> ''),
    CHECK (name <> ''),
    CHECK (timezone <> ''),
    CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE IF NOT EXISTS organisation_memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    member_ref text NOT NULL,
    role text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organisation_id, member_ref),
    CHECK (member_ref <> ''),
    CHECK (role <> '')
);

CREATE TABLE IF NOT EXISTS location_memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    member_ref text NOT NULL,
    role text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE CASCADE,
    UNIQUE (location_id, member_ref),
    CHECK (member_ref <> ''),
    CHECK (role <> '')
);

CREATE TABLE IF NOT EXISTS floors (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    slug text NOT NULL,
    name text NOT NULL,
    sort_order integer NOT NULL DEFAULT 0,
    canvas jsonb NOT NULL DEFAULT '{}'::jsonb,
    background_asset_ref text,
    is_active boolean NOT NULL DEFAULT true,
    version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE CASCADE,
    UNIQUE (location_id, slug),
    UNIQUE (id, organisation_id, location_id),
    CHECK (slug <> ''),
    CHECK (name <> ''),
    CHECK (sort_order >= 0),
    CHECK (version >= 1)
);

CREATE TABLE IF NOT EXISTS zones (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    floor_id uuid NOT NULL,
    name text NOT NULL,
    sort_order integer NOT NULL DEFAULT 0,
    is_active boolean NOT NULL DEFAULT true,
    version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (floor_id, organisation_id, location_id) REFERENCES floors (id, organisation_id, location_id) ON DELETE CASCADE,
    UNIQUE (floor_id, name),
    UNIQUE (id, organisation_id, location_id, floor_id),
    CHECK (name <> ''),
    CHECK (sort_order >= 0),
    CHECK (version >= 1)
);

CREATE TABLE IF NOT EXISTS tables (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    floor_id uuid NOT NULL,
    zone_id uuid NOT NULL,
    label text NOT NULL,
    capacity_label text NOT NULL,
    shape text NOT NULL,
    geometry jsonb NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    deleted_at timestamptz,
    version integer NOT NULL DEFAULT 1,
    legacy_table_id text UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (zone_id, organisation_id, location_id, floor_id) REFERENCES zones (id, organisation_id, location_id, floor_id) ON DELETE RESTRICT,
    UNIQUE (floor_id, label),
    UNIQUE (id, organisation_id, location_id),
    CHECK (label <> ''),
    CHECK (capacity_label <> ''),
    CHECK (shape IN ('rectangle', 'circle', 'square', 'custom')),
    CHECK (jsonb_typeof(geometry) = 'object'),
    CHECK (version >= 1),
    CHECK ((deleted_at IS NULL AND is_active = true) OR (deleted_at IS NOT NULL))
);

CREATE TABLE IF NOT EXISTS table_qr_capabilities (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    table_id uuid NOT NULL,
    token text NOT NULL UNIQUE,
    label text,
    issued_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz,
    revoked_at timestamptz,
    version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (table_id, organisation_id, location_id) REFERENCES tables (id, organisation_id, location_id) ON DELETE CASCADE,
    CHECK (token <> ''),
    CHECK (version >= 1),
    CHECK (expires_at IS NULL OR expires_at > issued_at)
);

CREATE TABLE IF NOT EXISTS table_occupancy (
    table_id uuid PRIMARY KEY,
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    status text NOT NULL DEFAULT 'available',
    current_session_id uuid,
    version integer NOT NULL DEFAULT 1,
    updated_by text,
    updated_at timestamptz NOT NULL DEFAULT now(),
    created_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (table_id, organisation_id, location_id) REFERENCES tables (id, organisation_id, location_id) ON DELETE CASCADE,
    UNIQUE (current_session_id),
    CHECK (status IN ('available', 'occupied')),
    CHECK (version >= 1),
    CHECK ((status = 'available' AND current_session_id IS NULL) OR (status = 'occupied' AND current_session_id IS NOT NULL))
);

CREATE TABLE IF NOT EXISTS table_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    table_id uuid NOT NULL,
    started_at timestamptz NOT NULL DEFAULT now(),
    ended_at timestamptz,
    party_size integer,
    opened_by text,
    closed_by text,
    source text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (table_id, organisation_id, location_id) REFERENCES tables (id, organisation_id, location_id) ON DELETE RESTRICT,
    CHECK (party_size IS NULL OR party_size > 0),
    CHECK (source <> ''),
    CHECK (ended_at IS NULL OR ended_at >= started_at)
);

CREATE UNIQUE INDEX IF NOT EXISTS table_sessions_one_active_per_table
    ON table_sessions (table_id)
    WHERE ended_at IS NULL;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'table_occupancy_current_session_fk'
    ) THEN
        ALTER TABLE table_occupancy
            ADD CONSTRAINT table_occupancy_current_session_fk
            FOREIGN KEY (current_session_id) REFERENCES table_sessions (id) ON DELETE RESTRICT
            DEFERRABLE INITIALLY DEFERRED;
    END IF;
END $$;

CREATE TABLE IF NOT EXISTS assist_requests (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    table_id uuid NOT NULL,
    table_session_id uuid,
    status text NOT NULL DEFAULT 'pending',
    requested_at timestamptz NOT NULL DEFAULT now(),
    acknowledged_at timestamptz,
    resolved_at timestamptz,
    cancelled_at timestamptz,
    requested_by text,
    acknowledged_by text,
    resolved_by text,
    cancelled_by text,
    source text NOT NULL,
    note text,
    version integer NOT NULL DEFAULT 1,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (table_id, organisation_id, location_id) REFERENCES tables (id, organisation_id, location_id) ON DELETE RESTRICT,
    FOREIGN KEY (table_session_id) REFERENCES table_sessions (id) ON DELETE SET NULL,
    CHECK (status IN ('pending', 'acknowledged', 'resolved', 'cancelled')),
    CHECK (source <> ''),
    CHECK (version >= 1),
    CHECK ((status = 'pending' AND acknowledged_at IS NULL AND resolved_at IS NULL AND cancelled_at IS NULL)
        OR (status = 'acknowledged' AND acknowledged_at IS NOT NULL AND resolved_at IS NULL AND cancelled_at IS NULL)
        OR (status = 'resolved' AND resolved_at IS NOT NULL AND cancelled_at IS NULL)
        OR (status = 'cancelled' AND cancelled_at IS NOT NULL AND resolved_at IS NULL))
);

CREATE INDEX IF NOT EXISTS assist_requests_active_by_table
    ON assist_requests (table_id, status, requested_at)
    WHERE status IN ('pending', 'acknowledged');

CREATE TABLE IF NOT EXISTS legacy_restaurant_mappings (
    legacy_restaurant_id text PRIMARY KEY,
    organisation_id uuid NOT NULL REFERENCES organisations (id) ON DELETE CASCADE,
    location_id uuid NOT NULL,
    migrated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (location_id, organisation_id) REFERENCES locations (id, organisation_id) ON DELETE CASCADE,
    UNIQUE (organisation_id, location_id),
    CHECK (legacy_restaurant_id <> '')
);

CREATE TABLE IF NOT EXISTS legacy_table_mappings (
    legacy_table_id text PRIMARY KEY,
    organisation_id uuid NOT NULL,
    location_id uuid NOT NULL,
    table_id uuid NOT NULL,
    migrated_at timestamptz NOT NULL DEFAULT now(),
    FOREIGN KEY (table_id, organisation_id, location_id) REFERENCES tables (id, organisation_id, location_id) ON DELETE CASCADE,
    UNIQUE (table_id),
    CHECK (legacy_table_id <> '')
);

CREATE INDEX IF NOT EXISTS floors_live_by_location ON floors (location_id, sort_order) WHERE is_active;
CREATE INDEX IF NOT EXISTS zones_live_by_floor ON zones (floor_id, sort_order) WHERE is_active;
CREATE INDEX IF NOT EXISTS tables_live_by_floor ON tables (floor_id, label) WHERE is_active AND deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS table_occupancy_by_location ON table_occupancy (location_id, status);
CREATE INDEX IF NOT EXISTS table_sessions_by_location_started ON table_sessions (location_id, started_at DESC);
