-- 000008_interaction_seed.down.sql
-- 清空交互规则种子数据。
DELETE FROM patient_contraindications;
DELETE FROM tag_interactions;
DELETE FROM class_interaction_rules;
DELETE FROM ingredient_interactions;
