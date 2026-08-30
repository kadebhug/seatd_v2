CREATE TABLE IF NOT EXISTS schema_migrations_marker (
    id integer PRIMARY KEY,
    applied_at timestamptz NOT NULL DEFAULT now()
);

INSERT INTO schema_migrations_marker (id)
VALUES (1)
ON CONFLICT (id) DO NOTHING;
