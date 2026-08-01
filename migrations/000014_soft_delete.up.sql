-- 000014_soft_delete.up.sql
-- 为缺少软删除的表增加 deleted_at 字段，确保所有删除操作可追溯审计。
ALTER TABLE drug_categories       ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE drug_interactions     ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE drug_ingredients      ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE ingredient_interactions ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE class_interaction_rules ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE tag_interactions      ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE suppliers             ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE drug_suppliers        ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE consultations         ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE adverse_reactions     ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE medication_guidances  ADD COLUMN deleted_at TIMESTAMPTZ;
ALTER TABLE clinical_services     ADD COLUMN deleted_at TIMESTAMPTZ;
