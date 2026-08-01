-- 000005_prescription_item_snapshot.down.sql
ALTER TABLE prescription_items DROP COLUMN IF EXISTS is_split_allowed;
ALTER TABLE prescription_items DROP COLUMN IF EXISTS retail_price;
