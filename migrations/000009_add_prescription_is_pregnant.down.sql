-- 000009_add_prescription_is_pregnant.down.sql
ALTER TABLE prescriptions DROP COLUMN IF EXISTS is_pregnant;
