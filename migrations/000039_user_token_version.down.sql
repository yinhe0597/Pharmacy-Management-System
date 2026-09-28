-- 000039_user_token_version.down.sql
DROP INDEX IF EXISTS idx_users_id_token_version;
ALTER TABLE users DROP COLUMN IF EXISTS token_version;
