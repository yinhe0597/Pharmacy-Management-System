-- 000007_add_interaction_engine.up.sql
-- 药物相互作用引擎：增强药品临床属性、成分/分类/标签级相互作用、患者禁忌与审核结果记录。

-- 1. 增强药品主表：临床属性
ALTER TABLE drugs ADD COLUMN active_ingredient     VARCHAR(200);    -- 标准化有效成分
ALTER TABLE drugs ADD COLUMN atc_code              VARCHAR(10);     -- ATC 编码
ALTER TABLE drugs ADD COLUMN pharmacological_group VARCHAR(100);    -- 药理分类（如 NSAID, ACEI, SSRI）
ALTER TABLE drugs ADD COLUMN pregnancy_category    CHAR(1);         -- FDA 妊娠分级 A/B/C/D/X
ALTER TABLE drugs ADD COLUMN age_min_years         SMALLINT;        -- 最低适用年龄
ALTER TABLE drugs ADD COLUMN age_max_years         SMALLINT;        -- 最高适用年龄（0=不限）
ALTER TABLE drugs ADD COLUMN interaction_tags      VARCHAR(500);    -- 交互标签，逗号分隔
ALTER TABLE drugs ADD COLUMN lactation_safe        BOOLEAN DEFAULT NULL; -- NULL=未知 true=安全 false=不安全
ALTER TABLE drugs ADD COLUMN contraindication_notes TEXT;           -- 禁忌症自由文本
CREATE INDEX idx_drugs_pharm_group ON drugs (pharmacological_group);

-- 2. 增强配伍禁忌表：循证信息
ALTER TABLE drug_interactions ADD COLUMN mechanism        VARCHAR(500);
ALTER TABLE drug_interactions ADD COLUMN evidence_level   CHAR(1) DEFAULT 'E';  -- A=Meta B=RCT C=观察 D=个案 E=专家经验
ALTER TABLE drug_interactions ADD COLUMN source_reference VARCHAR(500);
ALTER TABLE drug_interactions ADD COLUMN updated_at       TIMESTAMPTZ NOT NULL DEFAULT now();

-- 3. 药品-成分映射（支持复方制剂）
CREATE TABLE drug_ingredients (
    id              BIGSERIAL PRIMARY KEY,
    drug_id         BIGINT NOT NULL REFERENCES drugs(id) ON DELETE CASCADE,
    ingredient_name VARCHAR(200) NOT NULL,
    strength        VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_drug_ingredient UNIQUE (drug_id, ingredient_name)
);
CREATE INDEX idx_di_ingredient ON drug_ingredients (ingredient_name);
CREATE INDEX idx_di_drug       ON drug_ingredients (drug_id);

-- 4. 成分-成分相互作用规则
CREATE TABLE ingredient_interactions (
    id               BIGSERIAL PRIMARY KEY,
    ingredient_a     VARCHAR(200) NOT NULL,
    ingredient_b     VARCHAR(200) NOT NULL,
    level            SMALLINT NOT NULL,              -- 1禁忌 2慎用 3注意
    mechanism        VARCHAR(500),
    evidence_level   CHAR(1) DEFAULT 'C',
    source_reference VARCHAR(500),
    description      VARCHAR(500),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_ingredient_inter UNIQUE (ingredient_a, ingredient_b)
);
CREATE INDEX idx_ii_ingredient_a ON ingredient_interactions (ingredient_a);
CREATE INDEX idx_ii_ingredient_b ON ingredient_interactions (ingredient_b);

-- 5. 分类-分类相互作用规则
CREATE TABLE class_interaction_rules (
    id               BIGSERIAL PRIMARY KEY,
    class_a          VARCHAR(100) NOT NULL,
    class_b          VARCHAR(100) NOT NULL,
    level            SMALLINT NOT NULL,
    mechanism        VARCHAR(500),
    evidence_level   CHAR(1) DEFAULT 'C',
    source_reference VARCHAR(500),
    description      VARCHAR(500),
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_class_inter UNIQUE (class_a, class_b)
);
CREATE INDEX idx_ci_class_a ON class_interaction_rules (class_a);
CREATE INDEX idx_ci_class_b ON class_interaction_rules (class_b);

-- 6. 交互标签-标签相互作用规则
CREATE TABLE tag_interactions (
    id               BIGSERIAL PRIMARY KEY,
    tag_a            VARCHAR(50) NOT NULL,
    tag_b            VARCHAR(50) NOT NULL,
    level            SMALLINT NOT NULL,
    mechanism        VARCHAR(500),
    evidence_level   CHAR(1) DEFAULT 'C',
    source_reference VARCHAR(500),
    description      VARCHAR(500),
    is_active        BOOLEAN NOT NULL DEFAULT TRUE,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_tag_inter UNIQUE (tag_a, tag_b)
);

-- 7. 患者禁忌表（年龄、妊娠、哺乳、疾病等）
CREATE TABLE patient_contraindications (
    id                     BIGSERIAL PRIMARY KEY,
    drug_id                BIGINT REFERENCES drugs(id),
    ingredient             VARCHAR(200),
    contraindication_type  VARCHAR(50) NOT NULL,   -- age / pregnancy / lactation / disease / allergy
    condition_value        VARCHAR(200),            -- 条件值（如 "<12" / ">65" / "pregnant" / "renal_failure"）
    level                  SMALLINT NOT NULL DEFAULT 1,
    description            VARCHAR(500),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_pc_drug_id ON patient_contraindications (drug_id);
CREATE INDEX idx_pc_type    ON patient_contraindications (contraindication_type);

-- 8. 交互检测结果快照（审核追溯）
CREATE TABLE interaction_results (
    id                BIGSERIAL PRIMARY KEY,
    prescription_id   BIGINT NOT NULL REFERENCES prescriptions(id) ON DELETE CASCADE,
    strategy          VARCHAR(20) NOT NULL,         -- explicit / ingredient / class / tag
    drug_a_id         BIGINT REFERENCES drugs(id),
    drug_b_id         BIGINT REFERENCES drugs(id),
    drug_a_name       VARCHAR(100),
    drug_b_name       VARCHAR(100),
    level             SMALLINT NOT NULL,
    severity          VARCHAR(20) NOT NULL DEFAULT 'warning',  -- block / warning
    mechanism         VARCHAR(500),
    evidence_level    CHAR(1),
    resolved          BOOLEAN NOT NULL DEFAULT FALSE,
    resolved_by       BIGINT,
    resolved_remarks  VARCHAR(500),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_ir_prescription ON interaction_results (prescription_id);
