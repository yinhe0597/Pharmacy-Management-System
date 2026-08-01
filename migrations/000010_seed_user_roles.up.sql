-- 000010_seed_user_roles.up.sql
-- 扩展用户角色种子数据：医生、护士、药房主任、药师、调配、核对、采购、财务。
-- 使用 ON CONFLICT DO NOTHING 保证幂等，默认密码均为 admin123。

INSERT INTO users (username, password_hash, name, role, status) VALUES
    ('doctor',        '$2a$10$N12O3.KDfwSbtezh1s8jUuyu5.8ubDOx0L.YWUpxan9ny4Q/vACCu', '张医生',   'doctor',            1),
    ('nurse',         '$2a$10$N12O3.KDfwSbtezh1s8jUuyu5.8ubDOx0L.YWUpxan9ny4Q/vACCu', '王护士',   'nurse',             1),
    ('pharmacy_chief','$2a$10$N12O3.KDfwSbtezh1s8jUuyu5.8ubDOx0L.YWUpxan9ny4Q/vACCu', '李主任',   'pharmacy_director', 1),
    ('pharmacist',    '$2a$10$N12O3.KDfwSbtezh1s8jUuyu5.8ubDOx0L.YWUpxan9ny4Q/vACCu', '赵药师',   'pharmacist',        1),
    ('buyer',         '$2a$10$N12O3.KDfwSbtezh1s8jUuyu5.8ubDOx0L.YWUpxan9ny4Q/vACCu', '吴采购',   'buyer',             1),
    ('finance',       '$2a$10$N12O3.KDfwSbtezh1s8jUuyu5.8ubDOx0L.YWUpxan9ny4Q/vACCu', '陈财务',   'finance',           1)
ON CONFLICT (username) DO NOTHING;

-- 旧 dispenser/checker 角色停用（职能已合并到医生/药师）
UPDATE users SET status = 0 WHERE role IN ('dispenser', 'checker');
