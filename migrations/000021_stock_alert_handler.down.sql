-- 000021_stock_alert_handler.down.sql
ALTER TABLE stock_alerts DROP COLUMN IF EXISTS resolved_by;
ALTER TABLE stock_alerts DROP COLUMN IF EXISTS resolved_by_name;
