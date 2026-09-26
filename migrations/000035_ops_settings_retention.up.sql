-- 000035_ops_settings_retention.up.sql
-- 运维收口：库房可配置项 + 操作日志归档表 + 日志保留期设置。
--
-- 背景：
--   1) 收货/发药库房为代码硬编码（purchase_service.defaultReceiveLocation=1、
--      prescription_service.PrescriptionLocationID=2），多库房部署需改代码重发版；
--      本次迁入 system_settings，管理员可配（000030 机制复用）。
--   2) 全量写操作审计（middleware.AuditWrites）使 operation_logs 无限增长；
--      新增归档表 operation_logs_archive + 保留期设置 log_retention_days，
--      由调度器按月归档（INSERT 归档 + DELETE 原表，分批执行）。

-- 1) 新增系统设置项（幂等：INSERT ... ON CONFLICT DO NOTHING，重复执行安全）
INSERT INTO system_settings (key, value, description) VALUES
    ('default_receive_location', '1', '默认收货库房（inventory_locations.id，采购收货入库目标）'),
    ('default_dispense_location', '2', '默认发药库房（inventory_locations.id，处方发药来源）'),
    ('log_retention_days', '180', '操作日志保留天数（调度器归档早于此期限的记录，0=不归档）')
ON CONFLICT (key) DO NOTHING;

-- 2) 操作日志归档表（与 operation_logs 同构 + archived_at 落归档时间）
CREATE TABLE IF NOT EXISTS operation_logs_archive (
    id          BIGINT PRIMARY KEY,
    user_id     BIGINT,
    username    VARCHAR(50),
    user_role   VARCHAR(50),
    action      VARCHAR(50) NOT NULL,
    resource    VARCHAR(100) NOT NULL,
    resource_id BIGINT,
    method      VARCHAR(10),
    path        VARCHAR(200),
    ip          VARCHAR(50),
    detail      TEXT,
    created_at  TIMESTAMPTZ NOT NULL,
    archived_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_operation_logs_archive_created ON operation_logs_archive (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_operation_logs_archive_action ON operation_logs_archive (action);

-- 3) 原表 created_at 索引：归档范围删除（created_at < cutoff）走索引，避免全表扫描
CREATE INDEX IF NOT EXISTS idx_operation_logs_created ON operation_logs (created_at);

COMMENT ON TABLE operation_logs_archive IS '操作日志归档表：调度器将超保留期的 operation_logs 搬运至此（000035）';
