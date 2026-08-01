-- 000009_add_prescription_is_pregnant.up.sql
-- 处方新增 is_pregnant 字段，用于妊娠禁忌检查。
ALTER TABLE prescriptions ADD COLUMN is_pregnant BOOLEAN NOT NULL DEFAULT FALSE;
