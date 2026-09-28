-- 000038_soft_delete_unique_and_indexes.down.sql
-- 回滚：恢复全表唯一约束（注意：若已存在软删行与在用行同键，本回滚会失败——
--       这正是原缺陷的成因，恢复约束等于重新引入该限制）。
DROP INDEX IF EXISTS idx_charges_deleted;
DROP INDEX IF EXISTS idx_medical_records_deleted;
DROP INDEX IF EXISTS idx_visits_deleted;
DROP INDEX IF EXISTS idx_medication_guidances_deleted;
DROP INDEX IF EXISTS idx_adverse_reactions_deleted;
DROP INDEX IF EXISTS idx_consultations_deleted;
DROP INDEX IF EXISTS idx_clinical_services_deleted;
DROP INDEX IF EXISTS idx_tag_interactions_deleted;
DROP INDEX IF EXISTS idx_class_interaction_rules_deleted;
DROP INDEX IF EXISTS idx_ingredient_interactions_deleted;
DROP INDEX IF EXISTS idx_drug_interactions_deleted;
DROP INDEX IF EXISTS idx_drug_ingredients_deleted;
DROP INDEX IF EXISTS idx_drug_suppliers_deleted;
DROP INDEX IF EXISTS idx_users_deleted;
DROP INDEX IF EXISTS idx_suppliers_deleted;
DROP INDEX IF EXISTS idx_drug_categories_deleted;
DROP INDEX IF EXISTS idx_drugs_deleted;

DROP INDEX IF EXISTS uq_tag_inter_live;
DROP INDEX IF EXISTS uq_class_inter_live;
DROP INDEX IF EXISTS uq_ingredient_inter_live;
DROP INDEX IF EXISTS uq_interaction_live;
DROP INDEX IF EXISTS uq_drug_ingredient_live;
DROP INDEX IF EXISTS uq_drug_supplier_live;
DROP INDEX IF EXISTS uq_users_username_live;
DROP INDEX IF EXISTS uq_clinical_services_code_live;
DROP INDEX IF EXISTS uq_suppliers_code_live;
DROP INDEX IF EXISTS uq_drug_categories_code_live;
DROP INDEX IF EXISTS uq_drug_live;

ALTER TABLE tag_interactions          ADD CONSTRAINT uq_tag_inter          UNIQUE (tag_a, tag_b);
ALTER TABLE class_interaction_rules   ADD CONSTRAINT uq_class_inter        UNIQUE (class_a, class_b);
ALTER TABLE ingredient_interactions   ADD CONSTRAINT uq_ingredient_inter    UNIQUE (ingredient_a, ingredient_b);
ALTER TABLE drug_interactions         ADD CONSTRAINT uq_interaction         UNIQUE (drug_a_id, drug_b_id);
ALTER TABLE drug_ingredients          ADD CONSTRAINT uq_drug_ingredient      UNIQUE (drug_id, ingredient_name);
ALTER TABLE drug_suppliers            ADD CONSTRAINT uq_drug_supplier       UNIQUE (drug_id, supplier_id);
ALTER TABLE users                     ADD CONSTRAINT users_username_key     UNIQUE (username);
ALTER TABLE clinical_services         ADD CONSTRAINT clinical_services_code_key UNIQUE (code);
ALTER TABLE suppliers                 ADD CONSTRAINT suppliers_code_key     UNIQUE (code);
ALTER TABLE drug_categories           ADD CONSTRAINT drug_categories_code_key UNIQUE (code);
ALTER TABLE drugs                     ADD CONSTRAINT uq_drug                UNIQUE (generic_name, specification, manufacturer, dosage_form);
