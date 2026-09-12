-- 000033_stock_alert_dedup.down.sql
DROP INDEX IF EXISTS uq_stock_alerts_open;
ALTER TABLE stock_alerts ALTER COLUMN location_id DROP NOT NULL;
ALTER TABLE stock_alerts ALTER COLUMN batch_no DROP NOT NULL;
