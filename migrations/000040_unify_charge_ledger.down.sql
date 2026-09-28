-- 000040_unify_charge_ledger.down.sql
DROP INDEX IF EXISTS idx_charge_items_source;
ALTER TABLE charge_items DROP COLUMN IF EXISTS source_record_id;
DROP INDEX IF EXISTS idx_charge_records_visit;
ALTER TABLE charge_records DROP COLUMN IF EXISTS visit_id;
