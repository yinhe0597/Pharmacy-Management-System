-- 000028_prescription_item_route_group.down.sql
ALTER TABLE prescription_items DROP COLUMN IF EXISTS route;
ALTER TABLE prescription_items DROP COLUMN IF EXISTS batch_group;
