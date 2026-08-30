-- 专账/空安瓿手工登记允许不关联处方（prescription_id=0），
-- 外键约束会导致手工补录必然 500（SQLSTATE 23503），故移除弱关联外键，保留原索引。
ALTER TABLE special_drug_ledgers DROP CONSTRAINT IF EXISTS special_drug_ledgers_prescription_id_fkey;
ALTER TABLE ampoule_returns DROP CONSTRAINT IF EXISTS ampoule_returns_prescription_id_fkey;
