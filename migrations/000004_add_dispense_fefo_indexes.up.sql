-- 000004_add_dispense_fefo_indexes.up.sql
-- 1) 退药路径按处方明细查询发药记录，需 item_id 索引（模型已声明，补迁移）
-- 注：新版 000001 已含该索引，用 IF NOT EXISTS 保证全新部署幂等
CREATE INDEX IF NOT EXISTS idx_dispense_item ON prescription_dispense_records (item_id);

-- 2) FEFO 发药选批次热路径：按 (drug, location, is_split, expiry) 前缀建部分索引
CREATE INDEX IF NOT EXISTS idx_inventory_fefo ON inventory (drug_id, location_id, is_split, expiry_date) WHERE status = 1;
