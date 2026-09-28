-- 000040_unify_charge_ledger.up.sql
--
-- 统一记账口径：charges（结算单）+ charge_items（费用行）是**唯一记账凭证**；
-- charge_records 降级为「应收计费项目源」，只被结算单消费，不再被任何报表直接统计。
--
-- 根因：charge_records 没有 visit_id 列，无法回答「本次就诊花了多少钱」，
-- 于是 CalculateBill ② 只能用 (patient_id, created_at ∈ [挂号, 结束]) 的时间窗口
-- 启发式去猜归属——把同一患者其它就诊的计费项并入本次结算，且漏掉挂号前录入的项。
-- 而 charge_items 的 DDL 注释本就写明其 item_id 可指向 charge_records.id，
-- 说明原设计意图是「charge_records 为明细来源、charge_items 为凭证行」。
--
-- 本迁移：
--   1) charge_records 增加 visit_id，使结算单可按就诊精确归集，并回填历史处方来源行；
--   2) charge_items 增加 source_record_id，显式记录费用行对应的计费项目源，
--      消除 item_id 多态引用（可能指向 drugs.id / clinical_services.id / charge_records.id）
--      缺乏判别字段的歧义。

ALTER TABLE charge_records ADD COLUMN IF NOT EXISTS visit_id BIGINT REFERENCES visits(id);
CREATE INDEX IF NOT EXISTS idx_charge_records_visit ON charge_records (visit_id);

-- 历史回填：ref_type='prescription' 的行可经 prescriptions.visit_id 精确归集。
-- 其余行（手工录入）无就诊关联，保持 NULL，结算时不参与归集（不再落入时间窗口猜测）。
UPDATE charge_records cr
   SET visit_id = p.visit_id
  FROM prescriptions p
 WHERE cr.ref_type = 'prescription'
   AND cr.ref_id = p.id
   AND cr.visit_id IS NULL
   AND p.visit_id IS NOT NULL;

ALTER TABLE charge_items ADD COLUMN IF NOT EXISTS source_record_id BIGINT;
CREATE INDEX IF NOT EXISTS idx_charge_items_source ON charge_items (source_record_id);
