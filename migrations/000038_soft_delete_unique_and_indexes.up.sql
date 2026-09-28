-- 000038_soft_delete_unique_and_indexes.up.sql
--
-- 一、软删表上的「全局 UNIQUE」与软删语义冲突
--   000014 给 17 张表补了 deleted_at（软删），但 000001/000007 创建的唯一约束是
--   「全表」唯一。软删只是把行标记掉，行仍在表内，于是：
--     药品/分类/供应商/诊疗项目删除后，同一编码永远无法重建 → 落库撞 23505。
--   而多数创建路径的「是否重复」预检走默认作用域（deleted_at IS NULL）查不到软删行，
--   预检通过 → INSERT 撞唯一键 → 非业务错误 → 接口返回 500「系统异常」而非 409。
--   修法：改为部分唯一索引 WHERE deleted_at IS NULL（与 000031 charges、000033
--   stock_alerts 已验证过的范式一致），使编码在软删后可重建。
--
--   不转换的 4 个：visits.visit_no、charges.charge_no、medical_records.visit_id、
--   uq_drug(drugs.code 由应用层 GetByCode(Unscoped) 显式拦截以保历史可追溯)。
--   前两个是单据流水号，必须全局唯一、绝不可复用。
--
--   说明：既有数据必然满足更严的全局唯一，故转换后不会出现冲突，无需清洗。
--
-- 二、软删列零索引
--   model 中 17 个 DeletedAt 均声明 gorm:"index"，但 000001/000007/000014/000029
--   只 ADD COLUMN，没有任何一条 CREATE INDEX。每条 List/Find/Count 都隐式追加
--   WHERE deleted_at IS NULL，数据量上来后全部退化为顺序扫描。
--
-- 注：migrate.sh 以 --single-transaction 执行，无法使用 CREATE INDEX CONCURRENTLY。
--     本迁移涉及的均为小体量主数据/规则表，锁表窗口可接受。

-- ── 一、软删表唯一约束改部分唯一索引 ──────────────────────────────

-- 药品：一品一规一商
ALTER TABLE drugs DROP CONSTRAINT IF EXISTS uq_drug;
CREATE UNIQUE INDEX IF NOT EXISTS uq_drug_live
    ON drugs (generic_name, specification, manufacturer, dosage_form)
    WHERE deleted_at IS NULL;

-- 药品分类 / 供应商 / 诊疗项目 / 用户：编码
ALTER TABLE drug_categories DROP CONSTRAINT IF EXISTS drug_categories_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_drug_categories_code_live
    ON drug_categories (code) WHERE deleted_at IS NULL;

ALTER TABLE suppliers DROP CONSTRAINT IF EXISTS suppliers_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_suppliers_code_live
    ON suppliers (code) WHERE deleted_at IS NULL;

ALTER TABLE clinical_services DROP CONSTRAINT IF EXISTS clinical_services_code_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_clinical_services_code_live
    ON clinical_services (code) WHERE deleted_at IS NULL;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_key;
CREATE UNIQUE INDEX IF NOT EXISTS uq_users_username_live
    ON users (username) WHERE deleted_at IS NULL;

-- 配伍/成分/分类/标签交互规则与成分表
ALTER TABLE drug_suppliers DROP CONSTRAINT IF EXISTS uq_drug_supplier;
CREATE UNIQUE INDEX IF NOT EXISTS uq_drug_supplier_live
    ON drug_suppliers (drug_id, supplier_id) WHERE deleted_at IS NULL;

ALTER TABLE drug_ingredients DROP CONSTRAINT IF EXISTS uq_drug_ingredient;
CREATE UNIQUE INDEX IF NOT EXISTS uq_drug_ingredient_live
    ON drug_ingredients (drug_id, ingredient_name) WHERE deleted_at IS NULL;

ALTER TABLE drug_interactions DROP CONSTRAINT IF EXISTS uq_interaction;
CREATE UNIQUE INDEX IF NOT EXISTS uq_interaction_live
    ON drug_interactions (drug_a_id, drug_b_id) WHERE deleted_at IS NULL;

ALTER TABLE ingredient_interactions DROP CONSTRAINT IF EXISTS uq_ingredient_inter;
CREATE UNIQUE INDEX IF NOT EXISTS uq_ingredient_inter_live
    ON ingredient_interactions (ingredient_a, ingredient_b) WHERE deleted_at IS NULL;

ALTER TABLE class_interaction_rules DROP CONSTRAINT IF EXISTS uq_class_inter;
CREATE UNIQUE INDEX IF NOT EXISTS uq_class_inter_live
    ON class_interaction_rules (class_a, class_b) WHERE deleted_at IS NULL;

ALTER TABLE tag_interactions DROP CONSTRAINT IF EXISTS uq_tag_inter;
CREATE UNIQUE INDEX IF NOT EXISTS uq_tag_inter_live
    ON tag_interactions (tag_a, tag_b) WHERE deleted_at IS NULL;

-- ── 二、补齐缺失的 deleted_at 索引 ────────────────────────────────

CREATE INDEX IF NOT EXISTS idx_drugs_deleted              ON drugs (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_drug_categories_deleted     ON drug_categories (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_suppliers_deleted          ON suppliers (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_deleted              ON users (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_drug_suppliers_deleted     ON drug_suppliers (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_drug_ingredients_deleted   ON drug_ingredients (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_drug_interactions_deleted  ON drug_interactions (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_ingredient_interactions_deleted ON ingredient_interactions (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_class_interaction_rules_deleted  ON class_interaction_rules (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_tag_interactions_deleted   ON tag_interactions (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_clinical_services_deleted  ON clinical_services (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_consultations_deleted      ON consultations (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_adverse_reactions_deleted  ON adverse_reactions (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_medication_guidances_deleted ON medication_guidances (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_visits_deleted             ON visits (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_medical_records_deleted    ON medical_records (deleted_at) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_charges_deleted            ON charges (deleted_at) WHERE deleted_at IS NULL;
