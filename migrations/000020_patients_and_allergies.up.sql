-- 000020_patients_and_allergies.up.sql
-- 患者档案 + 过敏史（二期预留 IPatientService 完整实现）
-- 用途: 处方录入关联患者、审核时自动带入过敏史/哺乳史检查（docs/05 §2）

-- 患者档案
CREATE TABLE patients (
    id            BIGSERIAL PRIMARY KEY,
    card_no       VARCHAR(50)  NOT NULL,
    name          VARCHAR(50)  NOT NULL,
    gender        VARCHAR(10),
    age           VARCHAR(10),
    phone         VARCHAR(20),
    is_lactating  BOOLEAN NOT NULL DEFAULT FALSE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_patients_card_no UNIQUE (card_no)
);
CREATE INDEX idx_patients_name ON patients (name);
CREATE INDEX idx_patients_phone ON patients (phone);

-- 过敏史
CREATE TABLE patient_allergies (
    id          BIGSERIAL PRIMARY KEY,
    patient_id  BIGINT NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    drug_name   VARCHAR(100) NOT NULL,
    reaction    VARCHAR(200),
    severity    SMALLINT NOT NULL DEFAULT 1, -- 1轻 2中 3重
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_patient_allergies_patient ON patient_allergies (patient_id);

-- 处方增加哺乳期标记（与 is_pregnant 对称，审核时驱动哺乳期慎用检查）
ALTER TABLE prescriptions ADD COLUMN is_lactating BOOLEAN NOT NULL DEFAULT FALSE;
