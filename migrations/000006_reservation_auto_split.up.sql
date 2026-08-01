-- 000006_reservation_auto_split.up.sql
-- 自动拆零：预占记录标记「待拆整盒」。
-- need_split=true 表示该整盒预占在发药时需拆零发放；
-- split_units 为该盒拆开后用于处方的片数（其余 pack-split_units 片入拆零柜）。
ALTER TABLE stock_reservations ADD COLUMN need_split BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE stock_reservations ADD COLUMN split_units BIGINT NOT NULL DEFAULT 0;
