-- 000029_phase2_clinical.down.sql
ALTER TABLE prescriptions DROP COLUMN IF EXISTS visit_id;
DROP TABLE IF EXISTS charge_items;
DROP TABLE IF EXISTS charges;
DROP TABLE IF EXISTS medical_record_diagnoses;
DROP TABLE IF EXISTS medical_records;
DROP TABLE IF EXISTS visits;
