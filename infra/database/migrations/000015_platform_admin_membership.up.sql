ALTER TABLE organisations
    ADD COLUMN IF NOT EXISTS onboarded_at timestamptz;

INSERT INTO role_permissions (role_name, permission_name)
VALUES
    ('support', 'platform.admin'),
    ('support', 'audit.read')
ON CONFLICT (role_name, permission_name) DO NOTHING;

CREATE TABLE IF NOT EXISTS platform_memberships (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_profile_id uuid NOT NULL REFERENCES user_profiles (id) ON DELETE CASCADE,
    role text NOT NULL REFERENCES roles (name) ON DELETE RESTRICT,
    granted_by_actor_ref text NOT NULL,
    granted_at timestamptz NOT NULL DEFAULT now(),
    disabled_at timestamptz,
    disabled_by_actor_ref text,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CHECK (role IN ('platform_admin', 'support')),
    CHECK (granted_by_actor_ref <> ''),
    CHECK (disabled_by_actor_ref IS NULL OR disabled_by_actor_ref <> ''),
    CHECK (
        (disabled_at IS NULL AND disabled_by_actor_ref IS NULL)
        OR (disabled_at IS NOT NULL AND disabled_by_actor_ref IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS platform_memberships_active_user
    ON platform_memberships (user_profile_id)
    WHERE disabled_at IS NULL;

CREATE INDEX IF NOT EXISTS platform_memberships_user_profile
    ON platform_memberships (user_profile_id);

CREATE TABLE IF NOT EXISTS platform_admin_grants (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email text NOT NULL,
    role text NOT NULL REFERENCES roles (name) ON DELETE RESTRICT,
    invited_by_actor_ref text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    consumed_at timestamptz,
    consumed_by_user_profile_id uuid REFERENCES user_profiles (id) ON DELETE SET NULL,
    revoked_at timestamptz,
    revoked_by_actor_ref text,
    CHECK (btrim(email) <> ''),
    CHECK (role IN ('platform_admin', 'support')),
    CHECK (invited_by_actor_ref <> ''),
    CHECK (revoked_by_actor_ref IS NULL OR revoked_by_actor_ref <> ''),
    CHECK (
        (consumed_at IS NULL AND consumed_by_user_profile_id IS NULL)
        OR (consumed_at IS NOT NULL AND consumed_by_user_profile_id IS NOT NULL)
    ),
    CHECK (
        (revoked_at IS NULL AND revoked_by_actor_ref IS NULL)
        OR (revoked_at IS NOT NULL AND revoked_by_actor_ref IS NOT NULL)
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS platform_admin_grants_pending_email
    ON platform_admin_grants (lower(btrim(email)))
    WHERE consumed_at IS NULL AND revoked_at IS NULL;

INSERT INTO platform_memberships (
    user_profile_id,
    role,
    granted_by_actor_ref,
    granted_at,
    created_at,
    updated_at
)
SELECT DISTINCT ON (om.user_profile_id)
    om.user_profile_id,
    om.role,
    COALESCE(NULLIF(om.member_ref, ''), 'migration'),
    om.created_at,
    om.created_at,
    om.updated_at
FROM organisation_memberships om
WHERE om.role IN ('platform_admin', 'support')
  AND om.user_profile_id IS NOT NULL
  AND om.disabled_at IS NULL
ORDER BY om.user_profile_id, om.updated_at DESC
ON CONFLICT (user_profile_id) WHERE disabled_at IS NULL DO NOTHING;

DELETE FROM organisation_memberships
WHERE role IN ('platform_admin', 'support');

CREATE OR REPLACE FUNCTION platform_admin_lockout_guard()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM platform_memberships
        WHERE role = 'platform_admin'
          AND disabled_at IS NULL
    ) THEN
        RAISE EXCEPTION 'at least one active platform_admin is required'
            USING ERRCODE = '23514';
    END IF;
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS platform_admin_lockout ON platform_memberships;
CREATE CONSTRAINT TRIGGER platform_admin_lockout
    AFTER UPDATE OR DELETE ON platform_memberships
    DEFERRABLE INITIALLY DEFERRED
    FOR EACH ROW EXECUTE FUNCTION platform_admin_lockout_guard();

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'seatd_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON platform_memberships TO seatd_app;
        GRANT SELECT, INSERT, UPDATE, DELETE ON platform_admin_grants TO seatd_app;
    END IF;
END $$;

ALTER TABLE platform_memberships ENABLE ROW LEVEL SECURITY;
ALTER TABLE platform_memberships FORCE ROW LEVEL SECURITY;
ALTER TABLE platform_admin_grants ENABLE ROW LEVEL SECURITY;
ALTER TABLE platform_admin_grants FORCE ROW LEVEL SECURITY;

DROP POLICY IF EXISTS platform_memberships_platform_admin ON platform_memberships;
CREATE POLICY platform_memberships_platform_admin ON platform_memberships
    USING (seatd_is_platform_admin())
    WITH CHECK (seatd_is_platform_admin());

DROP POLICY IF EXISTS platform_admin_grants_platform_admin ON platform_admin_grants;
CREATE POLICY platform_admin_grants_platform_admin ON platform_admin_grants
    USING (seatd_is_platform_admin())
    WITH CHECK (seatd_is_platform_admin());
