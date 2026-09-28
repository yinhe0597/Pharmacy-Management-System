-- 000037_receipt_qc_result_default.up.sql
-- 修复收货质检门禁失效：purchase_receipt_items.qc_result 的 DEFAULT 1 会让
-- GORM/DB 把「未质检(0)」覆盖为「合格(1)」，导致 HasUninspected(qc_result NOT IN (1,2))
-- 恒为 false，收货单可在零质检下入库（违反 GSP「收货须质检合格方可入库」）。
--
-- 说明：此前已被默认成 1 的历史行无法区分「真合格」与「未登记」，此处不做回溯改写
-- （贸然改成 0 会把真实合格单误判为未质检并阻断已入库单据）；仅去掉列默认值，
-- 使迁移之后新登记的 qc_result 与客户端提交值严格一致。
ALTER TABLE purchase_receipt_items ALTER COLUMN qc_result DROP DEFAULT;
