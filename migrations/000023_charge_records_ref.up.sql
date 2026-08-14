-- 000023_charge_records_ref.up.sql
-- 计费闭环：计费记录关联来源单据（处方发药计费/退药冲正），支持幂等去重
ALTER TABLE charge_records ADD COLUMN ref_type VARCHAR(20);
ALTER TABLE charge_records ADD COLUMN ref_id BIGINT;
CREATE INDEX idx_charge_ref ON charge_records (ref_type, ref_id);
