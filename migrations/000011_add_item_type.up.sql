-- 000011_add_item_type.up.sql
-- 药房物品类型：药品(drug) vs 耗材(consumable)。
-- 诊疗项目（手法复位、静脉注射等）属于临床端，不在此列——见 docs/05 二期临床服务目录。
ALTER TABLE drugs ADD COLUMN item_type VARCHAR(20) NOT NULL DEFAULT 'drug';
COMMENT ON COLUMN drugs.item_type IS 'drug=药品 consumable=耗材（诊疗项目不入药房库存，由二期临床服务目录管理）';
