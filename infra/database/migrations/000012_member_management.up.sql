CREATE UNIQUE INDEX IF NOT EXISTS user_profiles_active_email_unique
    ON user_profiles (lower(btrim(email)))
    WHERE email IS NOT NULL AND status = 'active';
