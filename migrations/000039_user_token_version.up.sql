-- 000039_user_token_version.up.sql
--
-- JWT 无服务端吊销能力：改密、停用、角色变更后，已签发的 token 在 TTL（默认 720h=30 天）
-- 内仍然有效。checkActive 只能拦「已停用/已删」，拦不住「口令已轮换」。
-- 本迁移引入 token_version，签入 claims 并在每请求与库中当前值比对；
-- 口令轮换时自增该列，即可一次性作废该用户全部存量 token。
ALTER TABLE users ADD COLUMN IF NOT EXISTS token_version BIGINT NOT NULL DEFAULT 0;

-- 每请求复查状态/角色/口令版本都会读该行
CREATE INDEX IF NOT EXISTS idx_users_id_token_version ON users (id, token_version);
