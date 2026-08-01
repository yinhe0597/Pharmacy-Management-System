-- 000002_seed.up.sql 基础种子数据

-- 药品分类
INSERT INTO drug_categories (code, name, parent_id, sort_order) VALUES
    ('ANTIBIOTIC', '抗生素', 0, 1),
    ('ANTIVIRAL', '抗病毒药', 0, 2),
    ('CARDIO', '心血管用药', 0, 3),
    ('GI', '消化系统用药', 0, 4),
    ('RESP', '呼吸系统用药', 0, 5),
    ('CNS', '神经系统用药', 0, 6),
    ('NARCOTIC', '麻醉药品', 0, 7),
    ('PSYCHOTROPIC', '精神药品', 0, 8),
    ('OTHER', '其他', 0, 99);

-- 库存地点：药库 / 药房 / 麻精专柜（科室留待二期）
INSERT INTO inventory_locations (code, name, type, parent_id) VALUES
    ('WAREHOUSE', '中心药库', 1, 0),
    ('PHARMACY', '门诊药房', 2, 0),
    ('NARCOTIC_CABINET', '麻精专柜', 3, 0);

-- 默认管理员（密码：admin123，bcrypt 预生成）
INSERT INTO users (username, password_hash, name, role, status) VALUES
    ('admin', '$2a$10$N12O3.KDfwSbtezh1s8jUuyu5.8ubDOx0L.YWUpxan9ny4Q/vACCu', '系统管理员', 'admin', 1);
