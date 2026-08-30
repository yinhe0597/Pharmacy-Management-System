-- docs/20 S4 补强：同一就诊仅允许一张有效结算单（应用层幂等检查之外的数据库级约束）
CREATE UNIQUE INDEX IF NOT EXISTS uq_charges_visit_active
    ON charges (visit_id)
    WHERE deleted_at IS NULL;
