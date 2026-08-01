-- 000013_operation_logs.up.sql
-- 操作日志：记录所有关键业务操作（创建/更新/删除/审核等），供管理员审计。
CREATE TABLE operation_logs (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT,
    username    VARCHAR(50),
    user_role   VARCHAR(50),
    action      VARCHAR(50)  NOT NULL,         -- 操作动作：create/update/delete/review/login 等
    resource    VARCHAR(100) NOT NULL,          -- 操作资源：prescriptions/drugs/users 等
    resource_id BIGINT,                        -- 资源ID
    method      VARCHAR(10),                   -- HTTP method
    path        VARCHAR(200),                  -- 请求路径
    ip          VARCHAR(50),                   -- 客户端IP
    detail      TEXT,                          -- 操作详情
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ol_user_id    ON operation_logs (user_id);
CREATE INDEX idx_ol_created_at ON operation_logs (created_at DESC);
CREATE INDEX idx_ol_action     ON operation_logs (action);
CREATE INDEX idx_ol_resource   ON operation_logs (resource);
