-- 000003_alter_receipt_items.up.sql
-- 收货明细增加采购单明细关联，修复同一药品多行采购单的实收归集与超收校验
ALTER TABLE purchase_receipt_items ADD COLUMN order_item_id BIGINT;
CREATE INDEX idx_receipt_items_order ON purchase_receipt_items (order_item_id);
