-- 000028_prescription_item_route_group.up.sql
-- 开方精细化（docs/17 P3）：给药途径 + 分批配伍分组
ALTER TABLE prescription_items ADD COLUMN route VARCHAR(20);        -- oral/external/iv/im/iv_drip/inhale/other
ALTER TABLE prescription_items ADD COLUMN batch_group VARCHAR(20);  -- 分批组（如 口服组/输液组1）
