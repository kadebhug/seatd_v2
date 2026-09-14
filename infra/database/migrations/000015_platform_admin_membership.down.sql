DROP POLICY IF EXISTS platform_admin_grants_platform_admin ON platform_admin_grants;
DROP POLICY IF EXISTS platform_memberships_platform_admin ON platform_memberships;

DROP TRIGGER IF EXISTS platform_admin_lockout ON platform_memberships;
DROP FUNCTION IF EXISTS platform_admin_lockout_guard();

DROP TABLE IF EXISTS platform_admin_grants;
DROP TABLE IF EXISTS platform_memberships;

DELETE FROM role_permissions
WHERE role_name = 'support'
  AND permission_name IN ('platform.admin', 'audit.read');

ALTER TABLE organisations
    DROP COLUMN IF EXISTS onboarded_at;
