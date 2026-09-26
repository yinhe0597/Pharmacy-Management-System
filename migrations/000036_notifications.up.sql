-- 000036_notifications.up.sql
-- 站内通知：预警闭环的前端触达通道。
--
-- 背景：调度器只写 stock_alerts，值班人员需主动进系统查看；本次新增 notifications 表，
-- 调度器在生成预警后同步投递站内通知（广播 user_id IS NULL，或定向到具体用户），
-- 前端 header 铃铛展示未读数 + 下拉已读，形成「预警生成 → 通知触达 → 已读确认」闭环。
--
-- 设计：
--   user_id NULL = 全员广播（登录用户可见）；非 NULL = 定向通知。
--   level：info/warning/critical（critical 用于过期锁定等需立即处置项）。
--   resource/resource_id：点击跳转溯源（如 stock_alerts 明细），可空。
--   已读状态按用户隔离存放 notification_reads（广播行被一人已读不得影响他人未读数）。

CREATE TABLE IF NOT EXISTS notifications (
    id          BIGSERIAL PRIMARY KEY,
    user_id     BIGINT,
    title       VARCHAR(200) NOT NULL,
    content     VARCHAR(1000) NOT NULL DEFAULT '',
    level       VARCHAR(20) NOT NULL DEFAULT 'info',
    resource    VARCHAR(100) NOT NULL DEFAULT '',
    resource_id BIGINT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_notifications_user_created ON notifications (user_id, id DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_created ON notifications (created_at DESC);

CREATE TABLE IF NOT EXISTS notification_reads (
    notification_id BIGINT NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    user_id         BIGINT NOT NULL,
    read_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (notification_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_notification_reads_user ON notification_reads (user_id, notification_id);

COMMENT ON TABLE notifications IS '站内通知：预警/关键事件触达（000036）；user_id NULL=全员广播';
COMMENT ON TABLE notification_reads IS '通知已读回执（按用户隔离，广播已读不影响他人）';
