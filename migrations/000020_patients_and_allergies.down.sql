-- 000020_patients_and_allergies.down.sql
ALTER TABLE prescriptions DROP COLUMN IF EXISTS is_lactating;
DROP TABLE IF EXISTS patient_allergies;
DROP TABLE IF EXISTS patients;
