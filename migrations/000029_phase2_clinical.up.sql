-- 000029_phase2_clinical.up.sql
-- 二期就诊模块 S1（docs/20）：就诊 visits / 病历 medical_records(+diagnoses) / 合并结算 charges(+items)
-- 原则：金额一律 BIGINT 分；保留软删除列与一期风格一致；处方增加 visit_id 联动医嘱。

-- 1. 就诊
CREATE TABLE visits (
    id            BIGSERIAL PRIMARY KEY,
    visit_no      VARCHAR(30)  NOT NULL UNIQUE,          -- 就诊号（挂号单号）
    patient_id    BIGINT       NOT NULL REFERENCES patients(id),
    patient_name  VARCHAR(50)  NOT NULL,                 -- 姓名快照
    department    VARCHAR(50),                           -- 科室
    doctor_id     BIGINT       REFERENCES users(id),     -- 接诊医生（可为空=分诊后指定）
    doctor_name   VARCHAR(50),
    visit_type    VARCHAR(20)  NOT NULL DEFAULT 'outpatient', -- outpatient门诊/inpatient住院/refill出院带药
    status        VARCHAR(20)  NOT NULL DEFAULT 'waiting',    -- waiting待诊/visiting就诊中/finished已结束/cancelled退号
    registered_by VARCHAR(50),                           -- 挂号操作人
    registered_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    visited_at    TIMESTAMPTZ,                           -- 接诊时间
    finished_at   TIMESTAMPTZ,                           -- 结束时间
    remarks       VARCHAR(500),
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);
CREATE INDEX idx_visits_patient       ON visits (patient_id);
CREATE INDEX idx_visits_doctor        ON visits (doctor_id);
CREATE INDEX idx_visits_status        ON visits (status);
CREATE INDEX idx_visits_registered_at ON visits (registered_at);

-- 2. 病历（一就诊一病历，可更新；多诊断见 medical_record_diagnoses）
CREATE TABLE medical_records (
    id                 BIGSERIAL PRIMARY KEY,
    visit_id           BIGINT       NOT NULL REFERENCES visits(id) ON DELETE CASCADE,
    patient_id         BIGINT       NOT NULL REFERENCES patients(id),
    chief_complaint    TEXT,        -- 主诉
    present_illness    TEXT,        -- 现病史
    past_history       TEXT,        -- 既往史
    physical_exam      TEXT,        -- 体格检查
    temperature        NUMERIC(4,1),-- 体温 ℃
    systolic_pressure  INT,         -- 收缩压 mmHg
    diastolic_pressure INT,         -- 舒张压 mmHg
    pulse              INT,         -- 脉搏 次/分
    diagnosis          TEXT,        -- 诊断描述（自由文本）
    diagnosis_code     VARCHAR(10), -- 主诊断 ICD-10 编码（关联 diagnosis_codes.code）
    created_by         VARCHAR(50), -- 病历录入人
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ,
    CONSTRAINT uq_mr_visit UNIQUE (visit_id)
);
CREATE INDEX idx_mr_patient ON medical_records (patient_id);

-- 3. 病历多诊断（结构化，关联 ICD-10）
CREATE TABLE medical_record_diagnoses (
    id               BIGSERIAL PRIMARY KEY,
    medical_record_id BIGINT      NOT NULL REFERENCES medical_records(id) ON DELETE CASCADE,
    diagnosis_code   VARCHAR(10) NOT NULL,               -- ICD-10 编码
    diagnosis_name   VARCHAR(200),                       -- 诊断名称快照
    is_primary       BOOLEAN     NOT NULL DEFAULT FALSE, -- 主诊断
    sort_order       INT         NOT NULL DEFAULT 0,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_mrd_record ON medical_record_diagnoses (medical_record_id);

-- 4. 合并结算单（挂号费+诊查费+治疗费+检查费+药费合并）
CREATE TABLE charges (
    id              BIGSERIAL PRIMARY KEY,
    charge_no       VARCHAR(30)  NOT NULL UNIQUE,        -- 结算单号
    visit_id        BIGINT       NOT NULL REFERENCES visits(id),
    patient_id      BIGINT       NOT NULL REFERENCES patients(id),
    patient_name    VARCHAR(50)  NOT NULL,
    total_amount    BIGINT       NOT NULL DEFAULT 0,     -- 合计（分）
    discount_amount BIGINT       NOT NULL DEFAULT 0,     -- 优惠（分）
    payable_amount  BIGINT       NOT NULL DEFAULT 0,     -- 应收（分）
    paid_amount     BIGINT       NOT NULL DEFAULT 0,     -- 实收（分）
    status          VARCHAR(20)  NOT NULL DEFAULT 'pending', -- pending待收/paid已收/refunded已退
    operator_name   VARCHAR(50),
    paid_at         TIMESTAMPTZ,
    refunded_at     TIMESTAMPTZ,
    remarks         VARCHAR(500),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);
CREATE INDEX idx_charges_visit   ON charges (visit_id);
CREATE INDEX idx_charges_patient ON charges (patient_id);
CREATE INDEX idx_charges_status  ON charges (status);

-- 5. 结算单明细（多费用项）
CREATE TABLE charge_items (
    id         BIGSERIAL PRIMARY KEY,
    charge_id  BIGINT       NOT NULL REFERENCES charges(id) ON DELETE CASCADE,
    visit_id   BIGINT,                                  -- 冗余，便于按就诊查询
    item_type  VARCHAR(20)  NOT NULL,                   -- registration挂号/consultation诊查/treatment治疗/examination检查/drug药品/consumable耗材/clinical_service诊疗
    item_id    BIGINT,                                  -- 关联 clinical_services.id / drugs.id / charge_records.id
    item_name  VARCHAR(200) NOT NULL,                   -- 快照名称
    quantity   INT          NOT NULL DEFAULT 1,
    unit_price BIGINT       NOT NULL DEFAULT 0,         -- 单价（分）
    amount     BIGINT       NOT NULL DEFAULT 0,         -- 金额（分）
    sort_order INT          NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_ci_charge ON charge_items (charge_id);
CREATE INDEX idx_ci_visit  ON charge_items (visit_id);

-- 6. 处方联动就诊（医嘱归属就诊，S5 扩展 source 取值）
ALTER TABLE prescriptions ADD COLUMN visit_id BIGINT REFERENCES visits(id);
CREATE INDEX idx_prescriptions_visit ON prescriptions (visit_id);

COMMENT ON TABLE visits IS '二期就诊登记（docs/20 S1）';
COMMENT ON TABLE medical_records IS '就诊病历（一就诊一病历，多诊断见 medical_record_diagnoses）';
COMMENT ON TABLE charges IS '合并结算单：挂号/诊查/治疗/检查/药费合并收费';
COMMENT ON TABLE charge_items IS '结算单明细（多费用项）';
