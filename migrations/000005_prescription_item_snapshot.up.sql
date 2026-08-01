-- 000005_prescription_item_snapshot.up.sql
-- 混合发药支持：
-- 1) prescription_items.retail_price：盒价快照（整盒部分计价），混合发药时与 unit_price(拆零快照) 配合
-- 2) prescription_items.is_split_allowed：药品是否可拆零（用于分配器混合/整盒口径判定）
ALTER TABLE prescription_items ADD COLUMN retail_price BIGINT NOT NULL DEFAULT 0;
ALTER TABLE prescription_items ADD COLUMN is_split_allowed BOOLEAN NOT NULL DEFAULT FALSE;
