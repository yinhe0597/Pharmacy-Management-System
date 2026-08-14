-- 000022_drug_insurance_vbp.up.sql
-- 药品主数据 ↔ 医保/集采目录匹配标注（v1.3 参考数据收尾）
ALTER TABLE drugs ADD COLUMN insurance_class VARCHAR(4);   -- 甲类/乙类（来自 nhsa_drug_catalog）
ALTER TABLE drugs ADD COLUMN vbp_batch SMALLINT;           -- 集采批次（来自 vbp_drug_catalog）
