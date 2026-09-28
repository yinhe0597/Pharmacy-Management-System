-- 000037_receipt_qc_result_default.down.sql
ALTER TABLE purchase_receipt_items ALTER COLUMN qc_result SET DEFAULT 1;
