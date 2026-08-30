-- Local-only seed fixtures belong here. Do not add production data to seeds.

WITH org AS (
    INSERT INTO organisations (id, slug, name, legacy_restaurant_id)
    VALUES (
        '11111111-1111-1111-1111-111111111111',
        'demo-group',
        'Demo Restaurant Group',
        'legacy-demo-restaurant'
    )
    ON CONFLICT (id) DO UPDATE
    SET slug = EXCLUDED.slug,
        name = EXCLUDED.name,
        legacy_restaurant_id = EXCLUDED.legacy_restaurant_id,
        updated_at = now()
    RETURNING id
),
location AS (
    INSERT INTO locations (id, organisation_id, slug, name, timezone, legacy_restaurant_id)
    SELECT
        '22222222-2222-2222-2222-222222222222',
        id,
        'main-street',
        'Main Street',
        'Africa/Johannesburg',
        'legacy-demo-location'
    FROM org
    ON CONFLICT (id) DO UPDATE
    SET slug = EXCLUDED.slug,
        name = EXCLUDED.name,
        timezone = EXCLUDED.timezone,
        legacy_restaurant_id = EXCLUDED.legacy_restaurant_id,
        updated_at = now()
    RETURNING id, organisation_id
),
floor_ground AS (
    INSERT INTO floors (id, organisation_id, location_id, slug, name, sort_order, canvas)
    SELECT
        '33333333-3333-3333-3333-333333333333',
        organisation_id,
        id,
        'ground-floor',
        'Ground Floor',
        0,
        '{"width":1200,"height":800,"unit":"px"}'::jsonb
    FROM location
    ON CONFLICT (id) DO UPDATE
    SET name = EXCLUDED.name,
        canvas = EXCLUDED.canvas,
        updated_at = now()
    RETURNING id, organisation_id, location_id
),
floor_rooftop AS (
    INSERT INTO floors (id, organisation_id, location_id, slug, name, sort_order, canvas)
    SELECT
        '33333333-3333-3333-3333-333333333334',
        organisation_id,
        location_id,
        'rooftop',
        'Rooftop',
        1,
        '{"width":1000,"height":650,"unit":"px"}'::jsonb
    FROM floor_ground
    ON CONFLICT (id) DO UPDATE
    SET name = EXCLUDED.name,
        canvas = EXCLUDED.canvas,
        updated_at = now()
    RETURNING id, organisation_id, location_id
),
zone_dining AS (
    INSERT INTO zones (id, organisation_id, location_id, floor_id, name, sort_order)
    SELECT
        '44444444-4444-4444-4444-444444444441',
        organisation_id,
        location_id,
        id,
        'Dining Room',
        0
    FROM floor_ground
    ON CONFLICT (id) DO UPDATE
    SET name = EXCLUDED.name,
        sort_order = EXCLUDED.sort_order,
        updated_at = now()
    RETURNING id, organisation_id, location_id, floor_id
),
zone_patio AS (
    INSERT INTO zones (id, organisation_id, location_id, floor_id, name, sort_order)
    SELECT
        '44444444-4444-4444-4444-444444444442',
        organisation_id,
        location_id,
        id,
        'Patio',
        1
    FROM floor_ground
    ON CONFLICT (id) DO UPDATE
    SET name = EXCLUDED.name,
        sort_order = EXCLUDED.sort_order,
        updated_at = now()
    RETURNING id, organisation_id, location_id, floor_id
),
zone_rooftop AS (
    INSERT INTO zones (id, organisation_id, location_id, floor_id, name, sort_order)
    SELECT
        '44444444-4444-4444-4444-444444444443',
        organisation_id,
        location_id,
        id,
        'Rooftop Bar',
        0
    FROM floor_rooftop
    ON CONFLICT (id) DO UPDATE
    SET name = EXCLUDED.name,
        sort_order = EXCLUDED.sort_order,
        updated_at = now()
    RETURNING id, organisation_id, location_id, floor_id
),
seed_tables AS (
    INSERT INTO tables (
        id,
        organisation_id,
        location_id,
        floor_id,
        zone_id,
        label,
        capacity_label,
        shape,
        geometry,
        legacy_table_id
    )
    SELECT *
    FROM (
        SELECT
            '55555555-5555-5555-5555-555555555551'::uuid,
            organisation_id,
            location_id,
            floor_id,
            id,
            'T1',
            '4',
            'rectangle',
            '{"x":120,"y":140,"width":120,"height":80,"rotation":0}'::jsonb,
            'legacy-table-t1'
        FROM zone_dining
        UNION ALL
        SELECT
            '55555555-5555-5555-5555-555555555552'::uuid,
            organisation_id,
            location_id,
            floor_id,
            id,
            'T2',
            '2',
            'circle',
            '{"x":320,"y":140,"radius":46}'::jsonb,
            'legacy-table-t2'
        FROM zone_dining
        UNION ALL
        SELECT
            '55555555-5555-5555-5555-555555555553'::uuid,
            organisation_id,
            location_id,
            floor_id,
            id,
            'P1',
            '6',
            'rectangle',
            '{"x":140,"y":420,"width":160,"height":90,"rotation":0}'::jsonb,
            'legacy-table-p1'
        FROM zone_patio
        UNION ALL
        SELECT
            '55555555-5555-5555-5555-555555555554'::uuid,
            organisation_id,
            location_id,
            floor_id,
            id,
            'R1',
            '4',
            'square',
            '{"x":180,"y":180,"width":90,"height":90,"rotation":45}'::jsonb,
            'legacy-table-r1'
        FROM zone_rooftop
    ) rows (
        id,
        organisation_id,
        location_id,
        floor_id,
        zone_id,
        label,
        capacity_label,
        shape,
        geometry,
        legacy_table_id
    )
    ON CONFLICT (id) DO UPDATE
    SET label = EXCLUDED.label,
        capacity_label = EXCLUDED.capacity_label,
        shape = EXCLUDED.shape,
        geometry = EXCLUDED.geometry,
        updated_at = now()
    RETURNING id, organisation_id, location_id, label
)
INSERT INTO table_occupancy (table_id, organisation_id, location_id)
SELECT id, organisation_id, location_id
FROM seed_tables
ON CONFLICT (table_id) DO NOTHING;

INSERT INTO table_qr_capabilities (
    organisation_id,
    location_id,
    table_id,
    token,
    label,
    token_lookup_prefix,
    token_hash
)
SELECT
    organisation_id,
    location_id,
    id,
    left(full_token, 16),
    label || ' demo QR',
    left(full_token, 16),
    digest(full_token, 'sha256')
FROM (
    SELECT
        organisation_id,
        location_id,
        id,
        label,
        'demo-' || lower(label) || '-capability-token' AS full_token
    FROM tables
    WHERE organisation_id = '11111111-1111-1111-1111-111111111111'
) seeded
ON CONFLICT (token_hash) DO NOTHING;

INSERT INTO organisation_memberships (organisation_id, member_ref, role)
VALUES
    ('11111111-1111-1111-1111-111111111111', 'user:owner-demo', 'organisation_owner'),
    ('11111111-1111-1111-1111-111111111111', 'user:support-demo', 'read_only')
ON CONFLICT (organisation_id, member_ref) DO NOTHING;

INSERT INTO location_memberships (organisation_id, location_id, member_ref, role)
VALUES
    (
        '11111111-1111-1111-1111-111111111111',
        '22222222-2222-2222-2222-222222222222',
        'user:waiter-demo',
        'waiter'
    )
ON CONFLICT (location_id, member_ref) DO NOTHING;

INSERT INTO devices (
    id,
    organisation_id,
    location_id,
    device_type,
    platform,
    app_version,
    trust_state
)
VALUES (
    '66666666-6666-6666-6666-666666666666',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222',
    'waiter_mobile',
    'flutter-dev',
    '0.0.0-dev',
    'trusted'
)
ON CONFLICT (id) DO UPDATE
SET location_id = EXCLUDED.location_id,
    device_type = EXCLUDED.device_type,
    platform = EXCLUDED.platform,
    app_version = EXCLUDED.app_version,
    trust_state = EXCLUDED.trust_state,
    revoked_at = NULL,
    updated_at = now();

INSERT INTO legacy_restaurant_mappings (legacy_restaurant_id, organisation_id, location_id)
VALUES (
    'legacy-demo-restaurant',
    '11111111-1111-1111-1111-111111111111',
    '22222222-2222-2222-2222-222222222222'
)
ON CONFLICT (legacy_restaurant_id) DO NOTHING;

INSERT INTO legacy_table_mappings (legacy_table_id, organisation_id, location_id, table_id)
SELECT legacy_table_id, organisation_id, location_id, id
FROM tables
WHERE organisation_id = '11111111-1111-1111-1111-111111111111'
  AND legacy_table_id IS NOT NULL
ON CONFLICT (legacy_table_id) DO NOTHING;
