-- 000012_clinical_services.up.sql
-- 诊疗项目目录 + 计费记录。护士/医生可直接录入诊疗项目与耗材计费，不走处方流程。

-- 1. 诊疗项目目录
CREATE TABLE clinical_services (
    id          BIGSERIAL PRIMARY KEY,
    code        VARCHAR(20)  NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    category    VARCHAR(50),            -- 分类：手法治疗/注射操作/检查项目/材料费/其他
    unit_price  BIGINT       NOT NULL DEFAULT 0,  -- 单价（分）
    unit        VARCHAR(20)  NOT NULL DEFAULT '次', -- 计价单位
    status      SMALLINT     NOT NULL DEFAULT 1,   -- 1启用 0停用
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- 2. 计费记录（药品/耗材/诊疗项目统一计费入口）
CREATE TABLE charge_records (
    id              BIGSERIAL PRIMARY KEY,
    patient_name    VARCHAR(50)  NOT NULL,          -- 患者姓名
    patient_card_no VARCHAR(50),                    -- 患者卡号
    item_type       VARCHAR(20)  NOT NULL,          -- drug / consumable / clinical_service
    item_id         BIGINT,                         -- drugs.id 或 clinical_services.id
    item_name       VARCHAR(100) NOT NULL,          -- 快照名称
    quantity        INT          NOT NULL DEFAULT 1,
    unit_price      BIGINT       NOT NULL,          -- 单价（分）
    amount          BIGINT       NOT NULL,          -- 金额 = quantity × unit_price（分）
    operator_id     BIGINT,                         -- 操作人
    operator_name   VARCHAR(50),
    remarks         TEXT,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX idx_cr_patient ON charge_records (patient_name);
CREATE INDEX idx_cr_date    ON charge_records (created_at);
