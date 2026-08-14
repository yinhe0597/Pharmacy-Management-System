-- 000021_stock_alert_handler.up.sql
-- 预警处置闭环：记录处理人与处理时间，支持 resolved/ignored 状态流转
ALTER TABLE stock_alerts ADD COLUMN resolved_by BIGINT;
ALTER TABLE stock_alerts ADD COLUMN resolved_by_name VARCHAR(50);
