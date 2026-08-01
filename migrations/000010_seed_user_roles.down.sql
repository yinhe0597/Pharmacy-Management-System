-- 000010_seed_user_roles.down.sql
DELETE FROM users WHERE username IN ('doctor', 'nurse', 'pharmacy_chief', 'pharmacist', 'buyer', 'finance');
-- 恢复旧 dispenser/checker 用户
UPDATE users SET status = 1 WHERE role IN ('dispenser', 'checker');
