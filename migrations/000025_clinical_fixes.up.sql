-- 000025_clinical_fixes.up.sql
-- docs/15 复审修复：
--  1) 处方结构化诊断编码（ICD-10，G2）
--  2) 计费记录关联患者 + 红冲标记（G4/G5/G6）
ALTER TABLE prescriptions ADD COLUMN diagnosis_code VARCHAR(10);
CREATE INDEX idx_prescription_diagnosis ON prescriptions (diagnosis_code);

ALTER TABLE charge_records ADD COLUMN patient_id BIGINT;
ALTER TABLE charge_records ADD COLUMN voided BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX idx_charge_patient ON charge_records (patient_id);
