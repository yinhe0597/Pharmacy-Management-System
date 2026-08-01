-- 000010_seed_user_roles.down.sql
DELETE FROM users WHERE username IN ('doctor', 'nurse', 'pharmacy_chief', 'pharmacist', 'dispenser', 'checker', 'buyer', 'finance');
