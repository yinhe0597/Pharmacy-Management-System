-- 000035_ops_settings_retention.down.sql
DROP TABLE IF EXISTS operation_logs_archive;
DROP INDEX IF EXISTS idx_operation_logs_created;
DELETE FROM system_settings WHERE key IN ('default_receive_location', 'default_dispense_location', 'log_retention_days');
