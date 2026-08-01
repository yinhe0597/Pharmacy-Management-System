-- 000002_seed.down.sql 回滚种子数据
DELETE FROM users WHERE username = 'admin';
DELETE FROM inventory_locations WHERE code IN ('WAREHOUSE', 'PHARMACY', 'NARCOTIC_CABINET');
DELETE FROM drug_categories;
