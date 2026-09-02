CREATE TABLE IF NOT EXISTS web_sessions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_profile_id uuid NOT NULL REFERENCES user_profiles (id) ON DELETE CASCADE,
    lookup_prefix text NOT NULL,
    session_hash bytea NOT NULL,
    issued_at timestamptz NOT NULL DEFAULT now(),
    last_used_at timestamptz,
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz,
    rotated_from_session_id uuid REFERENCES web_sessions (id) ON DELETE SET NULL,
    user_agent text,
    ip_address inet,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    UNIQUE (lookup_prefix),
    CHECK (lookup_prefix <> ''),
    CHECK (length(session_hash) >= 32),
    CHECK (expires_at > issued_at)
);

CREATE INDEX IF NOT EXISTS web_sessions_user_active
    ON web_sessions (user_profile_id, expires_at DESC)
    WHERE revoked_at IS NULL;

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'seatd_app') THEN
        GRANT SELECT, INSERT, UPDATE, DELETE ON web_sessions TO seatd_app;
    END IF;
END $$;
