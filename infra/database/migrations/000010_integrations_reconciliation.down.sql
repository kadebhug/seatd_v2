DROP POLICY IF EXISTS integration_discrepancies_tenant_isolation ON integration_discrepancies;
DROP POLICY IF EXISTS integration_reconciliation_runs_tenant_isolation ON integration_reconciliation_runs;
DROP POLICY IF EXISTS integration_webhook_inbox_tenant_isolation ON integration_webhook_inbox;
DROP POLICY IF EXISTS integration_table_mappings_tenant_isolation ON integration_table_mappings;
DROP POLICY IF EXISTS integration_connections_tenant_isolation ON integration_connections;

DROP TABLE IF EXISTS integration_discrepancies;
DROP TABLE IF EXISTS integration_reconciliation_runs;
DROP TABLE IF EXISTS integration_webhook_inbox;
DROP TABLE IF EXISTS integration_table_mappings;
DROP TABLE IF EXISTS integration_connections;

DELETE FROM role_permissions
WHERE permission_name IN ('integrations.read', 'integrations.manage');

DELETE FROM permissions
WHERE name IN ('integrations.read', 'integrations.manage');
