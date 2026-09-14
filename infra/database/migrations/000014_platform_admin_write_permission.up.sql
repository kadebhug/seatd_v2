INSERT INTO permissions (name, description)
VALUES ('platform.admin.write', 'Mutate tenant state (suspend, reactivate, and future write actions) as a platform administrator')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name)
VALUES ('platform_admin', 'platform.admin.write')
ON CONFLICT (role_name, permission_name) DO NOTHING;
