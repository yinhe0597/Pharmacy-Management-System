-- 000026_refine_nurse_roles.up.sql
-- docs/18：护士角色细化为 跟诊护士 clinic_nurse 与 药房护士 pharmacy_nurse
-- 原 nurse 用户更名为 pharmacy_nurse 并承接药房侧职责；新增 clinic_nurse 承接诊室侧职责。
UPDATE users SET role = 'pharmacy_nurse', username = 'pharmacy_nurse'
WHERE username = 'nurse' AND role = 'nurse';

INSERT INTO users (username, password_hash, name, role, status) VALUES
    ('clinic_nurse', '$2a$10$N12O3.KDfwSbtezh1s8jUuyu5.8ubDOx0L.YWUpxan9ny4Q/vACCu', '孙跟诊护士', 'clinic_nurse', 1)
ON CONFLICT (username) DO NOTHING;
