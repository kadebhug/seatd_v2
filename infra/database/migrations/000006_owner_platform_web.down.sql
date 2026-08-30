ALTER TABLE locations
    DROP CONSTRAINT IF EXISTS locations_feature_flags_object,
    DROP CONSTRAINT IF EXISTS locations_operating_config_object;

ALTER TABLE locations
    DROP COLUMN IF EXISTS feature_flags,
    DROP COLUMN IF EXISTS operating_config;
