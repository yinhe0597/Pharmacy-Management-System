-- 000014_soft_delete.down.sql
ALTER TABLE drug_categories       DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE drug_interactions     DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE drug_ingredients      DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE ingredient_interactions DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE class_interaction_rules DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE tag_interactions      DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE suppliers             DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE drug_suppliers        DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE consultations         DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE adverse_reactions     DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE medication_guidances  DROP COLUMN IF EXISTS deleted_at;
ALTER TABLE clinical_services     DROP COLUMN IF EXISTS deleted_at;
