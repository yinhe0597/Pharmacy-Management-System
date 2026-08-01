-- 000007_add_interaction_engine.down.sql
-- 回滚药物相互作用引擎相关变更。

DROP TABLE IF EXISTS interaction_results;
DROP TABLE IF EXISTS patient_contraindications;
DROP TABLE IF EXISTS tag_interactions;
DROP TABLE IF EXISTS class_interaction_rules;
DROP TABLE IF EXISTS ingredient_interactions;
DROP TABLE IF EXISTS drug_ingredients;

ALTER TABLE drug_interactions DROP COLUMN IF EXISTS updated_at;
ALTER TABLE drug_interactions DROP COLUMN IF EXISTS source_reference;
ALTER TABLE drug_interactions DROP COLUMN IF EXISTS evidence_level;
ALTER TABLE drug_interactions DROP COLUMN IF EXISTS mechanism;

DROP INDEX IF EXISTS idx_drugs_pharm_group;
ALTER TABLE drugs DROP COLUMN IF EXISTS contraindication_notes;
ALTER TABLE drugs DROP COLUMN IF EXISTS lactation_safe;
ALTER TABLE drugs DROP COLUMN IF EXISTS interaction_tags;
ALTER TABLE drugs DROP COLUMN IF EXISTS age_max_years;
ALTER TABLE drugs DROP COLUMN IF EXISTS age_min_years;
ALTER TABLE drugs DROP COLUMN IF EXISTS pregnancy_category;
ALTER TABLE drugs DROP COLUMN IF EXISTS pharmacological_group;
ALTER TABLE drugs DROP COLUMN IF EXISTS atc_code;
ALTER TABLE drugs DROP COLUMN IF EXISTS active_ingredient;
