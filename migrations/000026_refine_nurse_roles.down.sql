-- 000026_refine_nurse_roles.down.sql
DELETE FROM users WHERE username = 'clinic_nurse';
UPDATE users SET role = 'nurse', username = 'nurse' WHERE username = 'pharmacy_nurse';
