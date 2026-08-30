-- 恢复外键（回滚用）
ALTER TABLE special_drug_ledgers
    ADD CONSTRAINT special_drug_ledgers_prescription_id_fkey
    FOREIGN KEY (prescription_id) REFERENCES prescriptions(id);
ALTER TABLE ampoule_returns
    ADD CONSTRAINT ampoule_returns_prescription_id_fkey
    FOREIGN KEY (prescription_id) REFERENCES prescriptions(id);
