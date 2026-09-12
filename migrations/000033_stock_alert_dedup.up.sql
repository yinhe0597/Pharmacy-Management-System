-- 000033_stock_alert_dedup.up.sql
-- 预警去重加固（审计项 L6）：
--   1) 归一 location_id / batch_no（NULL → 0 / ''），使去重键可用普通列唯一索引表达；
--   2) 清理历史重复的 open 预警（同键保留 id 最小的一条），否则无法建唯一索引；
--   3) 建立「同一 类型+药品+库房+批次 在 open 状态下唯一」的部分唯一索引，
--      配合仓储层 ON CONFLICT DO NOTHING，杜绝调度器并发/重入产生重复预警。
--
-- 背景：原实现为 HasOpenByKey 预查询 + Create 两步（非原子），且「低于下限」预警
-- 以字面量 'ALL' 查询、却以空批次写入，去重键完全错配 → 每次调度都会新增一条重复预警。

UPDATE stock_alerts SET location_id = 0 WHERE location_id IS NULL;
ALTER TABLE stock_alerts ALTER COLUMN location_id SET DEFAULT 0;
ALTER TABLE stock_alerts ALTER COLUMN location_id SET NOT NULL;

UPDATE stock_alerts SET batch_no = '' WHERE batch_no IS NULL;
ALTER TABLE stock_alerts ALTER COLUMN batch_no SET DEFAULT '';
ALTER TABLE stock_alerts ALTER COLUMN batch_no SET NOT NULL;

-- 历史重复清理（仅 open 状态参与去重）
DELETE FROM stock_alerts a
USING stock_alerts b
WHERE a.id > b.id
  AND a.status = 'open'
  AND b.status = 'open'
  AND a.alert_type = b.alert_type
  AND a.drug_id = b.drug_id
  AND a.location_id = b.location_id
  AND a.batch_no = b.batch_no;

CREATE UNIQUE INDEX IF NOT EXISTS uq_stock_alerts_open
    ON stock_alerts (alert_type, drug_id, location_id, batch_no)
    WHERE status = 'open';
