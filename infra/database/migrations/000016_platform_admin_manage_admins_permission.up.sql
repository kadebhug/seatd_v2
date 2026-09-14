INSERT INTO permissions (name, description)
VALUES ('platform.admin.manage_admins', 'Invite, revoke, and reactivate platform administrator accounts')
ON CONFLICT (name) DO NOTHING;

INSERT INTO role_permissions (role_name, permission_name)
VALUES ('platform_admin', 'platform.admin.manage_admins')
ON CONFLICT (role_name, permission_name) DO NOTHING;
