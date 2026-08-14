-- 000022_drug_insurance_vbp.down.sql
ALTER TABLE drugs DROP COLUMN IF EXISTS insurance_class;
ALTER TABLE drugs DROP COLUMN IF EXISTS vbp_batch;
