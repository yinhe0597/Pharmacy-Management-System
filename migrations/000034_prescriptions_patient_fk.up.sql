-- 000034_prescriptions_patient_fk.up.sql
-- L2 收尾：prescriptions.patient_id 落数据库外键。
--
-- 背景：该列自 000001 起为可空 BIGINT，但一直只有服务层校验患者存在（应用层保证），
-- 数据库层无外键约束；且历史数据用 0 表示「未关联患者」，与外键语义冲突。
--
-- 处理：
--   1) 把 0 / 指向不存在患者的值规范化为 NULL（0 永远不匹配 patients.id）；
--   2) 补外键 ON DELETE SET NULL——患者被删除时处方保留、仅解除关联
--      （处方是医疗文书，不得随患者档案级联删除）。
ALTER TABLE prescriptions
    ALTER COLUMN patient_id DROP DEFAULT;

UPDATE prescriptions p
SET patient_id = NULL
WHERE p.patient_id IS NOT NULL
  AND NOT EXISTS (SELECT 1 FROM patients pt WHERE pt.id = p.patient_id);

ALTER TABLE prescriptions
    ADD CONSTRAINT fk_prescriptions_patient
    FOREIGN KEY (patient_id) REFERENCES patients(id) ON DELETE SET NULL;

COMMENT ON COLUMN prescriptions.patient_id IS '关联患者档案（可空，NULL=未关联）；FK → patients(id) ON DELETE SET NULL';
