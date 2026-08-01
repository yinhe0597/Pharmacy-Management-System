-- 000006_reservation_auto_split.down.sql
ALTER TABLE stock_reservations DROP COLUMN IF EXISTS split_units;
ALTER TABLE stock_reservations DROP COLUMN IF EXISTS need_split;
