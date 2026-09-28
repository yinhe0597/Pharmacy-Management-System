-- 000041_doc_segments.up.sql
--
-- 业务单号的多副本安全性（此前 seq.Next 为「前缀+Unix秒+进程内3位自增」，
-- 只在单进程内唯一；多副本部署时同秒同序号必然撞 UNIQUE 约束，
-- 而 PostgreSQL 下唯一冲突会把事务置入 aborted(25P02)，整笔业务回滚）。
--
-- 改为号段表：prefix → 已分配到的绝对序号上界。
-- 每次用一条 INSERT ... ON CONFLICT DO UPDATE ... RETURNING 原子申请一段号
-- （默认 256 个），该区间由本进程独占，与其它副本不相交 → 全局唯一。
-- DB 往返摊薄到 1/256 次。
--
-- 单号格式：<prefix><yyyyMMddHHmmss><8位绝对序号>
--   · 前缀 + 时间戳保持「单号可按时间排序」这一药房现场运维刚需；
--   · 8 位绝对序号保证跨副本全局唯一（同一 prefix 下由号段表单调分配）。
--
-- 与旧格式（前缀+10位秒+3位序号 = 前缀后 13 位）长度不同，故不可能与存量单号冲突。

CREATE TABLE IF NOT EXISTS doc_segments (
    prefix     VARCHAR(10) PRIMARY KEY,
    next_val   BIGINT      NOT NULL DEFAULT 0,  -- 已分配到的绝对序号上界（不含）
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMENT ON TABLE doc_segments IS '业务单号号段表：各前缀的绝对序号分配器，保证多副本下全局唯一';
