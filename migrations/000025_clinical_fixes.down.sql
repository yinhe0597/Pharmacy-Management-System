-- 000025_clinical_fixes.down.sql
DROP INDEX IF EXISTS idx_charge_patient;
ALTER TABLE charge_records DROP COLUMN IF EXISTS patient_id;
ALTER TABLE charge_records DROP COLUMN IF EXISTS voided;
DROP INDEX IF EXISTS idx_prescription_diagnosis;
ALTER TABLE prescriptions DROP COLUMN IF EXISTS diagnosis_code;
