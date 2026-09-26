-- 000034_prescriptions_patient_fk.down.sql
ALTER TABLE prescriptions DROP CONSTRAINT IF EXISTS fk_prescriptions_patient;
