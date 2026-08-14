-- 000023_charge_records_ref.down.sql
DROP INDEX IF EXISTS idx_charge_ref;
ALTER TABLE charge_records DROP COLUMN IF EXISTS ref_type;
ALTER TABLE charge_records DROP COLUMN IF EXISTS ref_id;
