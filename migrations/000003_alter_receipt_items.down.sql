-- 000003_alter_receipt_items.down.sql
DROP INDEX IF EXISTS idx_receipt_items_order;
ALTER TABLE purchase_receipt_items DROP COLUMN IF EXISTS order_item_id;
