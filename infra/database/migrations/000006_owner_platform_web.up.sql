ALTER TABLE locations
    ADD COLUMN IF NOT EXISTS operating_config jsonb NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS feature_flags jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE locations
    DROP CONSTRAINT IF EXISTS locations_operating_config_object,
    DROP CONSTRAINT IF EXISTS locations_feature_flags_object;

ALTER TABLE locations
    ADD CONSTRAINT locations_operating_config_object
    CHECK (jsonb_typeof(operating_config) = 'object'),
    ADD CONSTRAINT locations_feature_flags_object
    CHECK (jsonb_typeof(feature_flags) = 'object');
