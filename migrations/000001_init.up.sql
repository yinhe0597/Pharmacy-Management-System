-- 000001_init.up.sql 药房管理系统初始建表
-- 金额字段均为 bigint（分）；数量按最小发药单位口径，见 docs/02 §0

-- 1. 系统用户
CREATE TABLE users (
    id            BIGSERIAL PRIMARY KEY,
    username      VARCHAR(50)  NOT NULL UNIQUE,
    password_hash VARCHAR(100) NOT NULL,
    name          VARCHAR(50)  NOT NULL,
    role          VARCHAR(20)  NOT NULL,
    phone         VARCHAR(20),
    status        SMALLINT     NOT NULL DEFAULT 1,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at    TIMESTAMPTZ
);
COMMENT ON COLUMN users.role IS 'admin/pharmacist/dispenser/checker/buyer';

-- 2. 药品分类
CREATE TABLE drug_categories (
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(20)  NOT NULL UNIQUE,
    name       VARCHAR(50)  NOT NULL,
    parent_id  BIGINT       NOT NULL DEFAULT 0,
    sort_order INT          NOT NULL DEFAULT 0,
    status     SMALLINT     NOT NULL DEFAULT 1,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- 3. 药品主数据
CREATE TABLE drugs (
    id                   BIGSERIAL PRIMARY KEY,
    code                 VARCHAR(20)  NOT NULL UNIQUE,
    generic_name         VARCHAR(100) NOT NULL,
    brand_name           VARCHAR(100),
    dosage_form          VARCHAR(30)  NOT NULL,
    specification        VARCHAR(100) NOT NULL,
    manufacturer         VARCHAR(100) NOT NULL,
    approval_number      VARCHAR(50),
    barcode              VARCHAR(50),
    category_id          BIGINT       REFERENCES drug_categories(id),
    base_unit            VARCHAR(20)  NOT NULL,
    split_unit           VARCHAR(20),
    pack_size            INT          NOT NULL DEFAULT 1,
    is_split_allowed     BOOLEAN      NOT NULL DEFAULT FALSE,
    retail_price         BIGINT       NOT NULL DEFAULT 0,
    purchase_price       BIGINT       NOT NULL DEFAULT 0,
    split_retail_price   BIGINT       NOT NULL DEFAULT 0,
    split_purchase_price BIGINT       NOT NULL DEFAULT 0,
    antibiotic_level     SMALLINT     NOT NULL DEFAULT 0,
    special_control_type SMALLINT     NOT NULL DEFAULT 0,
    psychotropic_level   SMALLINT     NOT NULL DEFAULT 0,
    max_single_dose      BIGINT,
    max_daily_dose       BIGINT,
    expiry_warning_days  INT          NOT NULL DEFAULT 90,
    py_code              VARCHAR(50),
    status               SMALLINT     NOT NULL DEFAULT 1,
    is_frozen            BOOLEAN      NOT NULL DEFAULT FALSE,
    remarks              TEXT,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT now(),
    deleted_at           TIMESTAMPTZ,
    CONSTRAINT uq_drug UNIQUE (generic_name, specification, manufacturer, dosage_form)
);
CREATE INDEX idx_drugs_generic_name ON drugs (generic_name);
CREATE INDEX idx_drugs_py_code     ON drugs (py_code);
CREATE INDEX idx_drugs_category    ON drugs (category_id);
CREATE INDEX idx_drugs_special     ON drugs (special_control_type);

-- 4. 药品配伍禁忌
CREATE TABLE drug_interactions (
    id          BIGSERIAL PRIMARY KEY,
    drug_a_id   BIGINT NOT NULL REFERENCES drugs(id),
    drug_b_id   BIGINT NOT NULL REFERENCES drugs(id),
    level       SMALLINT NOT NULL,
    description VARCHAR(500),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_interaction UNIQUE (drug_a_id, drug_b_id)
);
CREATE INDEX idx_interaction_a ON drug_interactions (drug_a_id);
CREATE INDEX idx_interaction_b ON drug_interactions (drug_b_id);

-- 5. 供应商
CREATE TABLE suppliers (
    id               BIGSERIAL PRIMARY KEY,
    code             VARCHAR(20)  NOT NULL UNIQUE,
    name             VARCHAR(100) NOT NULL,
    contact_person   VARCHAR(50),
    phone            VARCHAR(20),
    address          VARCHAR(200),
    qualification_no VARCHAR(50),
    status           SMALLINT NOT NULL DEFAULT 1,
    remarks          TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ
);

-- 6. 药品-供应商供货关系
CREATE TABLE drug_suppliers (
    id               BIGSERIAL PRIMARY KEY,
    drug_id          BIGINT NOT NULL REFERENCES drugs(id),
    supplier_id      BIGINT NOT NULL REFERENCES suppliers(id),
    is_default       BOOLEAN NOT NULL DEFAULT FALSE,
    purchase_price   BIGINT,
    last_purchase_at TIMESTAMPTZ,
    status           SMALLINT NOT NULL DEFAULT 1,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_drug_supplier UNIQUE (drug_id, supplier_id)
);

-- 7. 库存地点
CREATE TABLE inventory_locations (
    id         BIGSERIAL PRIMARY KEY,
    code       VARCHAR(20) NOT NULL UNIQUE,
    name       VARCHAR(50) NOT NULL,
    type       SMALLINT    NOT NULL,
    parent_id  BIGINT      NOT NULL DEFAULT 0,
    is_active  BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 8. 库存上下限设置（拆零单位 LDU）
CREATE TABLE drug_stock_settings (
    id            BIGSERIAL PRIMARY KEY,
    drug_id       BIGINT NOT NULL REFERENCES drugs(id),
    location_id   BIGINT NOT NULL REFERENCES inventory_locations(id),
    min_quantity  BIGINT NOT NULL DEFAULT 0,
    max_quantity  BIGINT NOT NULL DEFAULT 0,
    reorder_qty   BIGINT NOT NULL DEFAULT 0,
    is_enabled    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_stock_setting UNIQUE (drug_id, location_id)
);

-- 9. 库存（核心）
CREATE TABLE inventory (
    id                BIGSERIAL PRIMARY KEY,
    drug_id           BIGINT NOT NULL REFERENCES drugs(id),
    location_id       BIGINT NOT NULL REFERENCES inventory_locations(id),
    batch_no          VARCHAR(50) NOT NULL,
    expiry_date       DATE        NOT NULL,
    quantity          BIGINT      NOT NULL DEFAULT 0,
    reserved_quantity BIGINT      NOT NULL DEFAULT 0,
    is_split          BOOLEAN     NOT NULL DEFAULT FALSE,
    unit_price        BIGINT      NOT NULL DEFAULT 0,
    received_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    status            SMALLINT    NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_inventory UNIQUE (drug_id, location_id, batch_no, expiry_date, is_split)
);
CREATE INDEX idx_inventory_drug_loc  ON inventory (drug_id, location_id);
CREATE INDEX idx_inventory_expiry    ON inventory (expiry_date) WHERE status = 1;
CREATE INDEX idx_inventory_available ON inventory (drug_id, location_id, expiry_date) WHERE status = 1;
CREATE INDEX idx_inventory_fefo      ON inventory (drug_id, location_id, is_split, expiry_date) WHERE status = 1;

-- 10. 库存流水
CREATE TABLE inventory_transactions (
    id               BIGSERIAL PRIMARY KEY,
    transaction_no   VARCHAR(30) NOT NULL UNIQUE,
    drug_id          BIGINT NOT NULL REFERENCES drugs(id),
    location_id      BIGINT NOT NULL REFERENCES inventory_locations(id),
    batch_no         VARCHAR(50),
    expiry_date      DATE,
    quantity         BIGINT NOT NULL,
    is_split         BOOLEAN NOT NULL DEFAULT FALSE,
    before_quantity  BIGINT,
    after_quantity   BIGINT,
    txn_type         VARCHAR(20) NOT NULL,
    ref_type         VARCHAR(30),
    ref_id           BIGINT,
    operator_id      BIGINT,
    operator_name    VARCHAR(50),
    remarks          VARCHAR(200),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_inv_txn_drug_time ON inventory_transactions (drug_id, created_at);
CREATE INDEX idx_inv_txn_ref       ON inventory_transactions (ref_type, ref_id);
CREATE INDEX idx_inv_txn_type      ON inventory_transactions (txn_type);

-- 11. 库存预占记录
CREATE TABLE stock_reservations (
    id             BIGSERIAL PRIMARY KEY,
    reservation_no VARCHAR(30) NOT NULL UNIQUE,
    ref_type       VARCHAR(30) NOT NULL,
    ref_id         BIGINT NOT NULL,
    item_id        BIGINT,
    inventory_id   BIGINT NOT NULL REFERENCES inventory(id),
    drug_id        BIGINT NOT NULL,
    location_id    BIGINT NOT NULL,
    batch_no       VARCHAR(50) NOT NULL,
    expiry_date    DATE NOT NULL,
    is_split       BOOLEAN NOT NULL DEFAULT FALSE,
    quantity       BIGINT NOT NULL,
    status         VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    released_at    TIMESTAMPTZ
);
CREATE INDEX idx_reservation_ref ON stock_reservations (ref_type, ref_id, status);
CREATE INDEX idx_reservation_inv ON stock_reservations (inventory_id, status);

-- 12. 采购单
CREATE TABLE purchase_orders (
    id            BIGSERIAL PRIMARY KEY,
    purchase_no   VARCHAR(30) NOT NULL UNIQUE,
    supplier_id   BIGINT NOT NULL REFERENCES suppliers(id),
    status        VARCHAR(20) NOT NULL DEFAULT 'draft',
    expected_at   DATE,
    total_amount  BIGINT NOT NULL DEFAULT 0,
    created_by    BIGINT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    received_at   TIMESTAMPTZ,
    remarks       TEXT
);
CREATE INDEX idx_purchase_supplier ON purchase_orders (supplier_id, status);

-- 13. 采购单明细
CREATE TABLE purchase_order_items (
    id                BIGSERIAL PRIMARY KEY,
    purchase_order_id BIGINT NOT NULL REFERENCES purchase_orders(id) ON DELETE CASCADE,
    drug_id           BIGINT NOT NULL REFERENCES drugs(id),
    quantity          BIGINT NOT NULL,
    unit_price        BIGINT NOT NULL,
    amount            BIGINT NOT NULL DEFAULT 0,
    received_quantity BIGINT NOT NULL DEFAULT 0
);
CREATE INDEX idx_po_items ON purchase_order_items (purchase_order_id);

-- 14. 采购收货单
CREATE TABLE purchase_receipts (
    id                BIGSERIAL PRIMARY KEY,
    receipt_no        VARCHAR(30) NOT NULL UNIQUE,
    purchase_order_id BIGINT REFERENCES purchase_orders(id),
    supplier_id       BIGINT NOT NULL REFERENCES suppliers(id),
    status            VARCHAR(20) NOT NULL DEFAULT 'pending_quality',
    total_amount      BIGINT NOT NULL DEFAULT 0,
    received_at       TIMESTAMPTZ,
    received_by       BIGINT,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 15. 收货明细（批次绑定）
CREATE TABLE purchase_receipt_items (
    id                BIGSERIAL PRIMARY KEY,
    receipt_id        BIGINT NOT NULL REFERENCES purchase_receipts(id) ON DELETE CASCADE,
    drug_id           BIGINT NOT NULL REFERENCES drugs(id),
    ordered_quantity  BIGINT NOT NULL,
    received_quantity BIGINT NOT NULL,
    batch_no          VARCHAR(50) NOT NULL,
    expiry_date       DATE NOT NULL,
    unit_price        BIGINT NOT NULL,
    qc_result         SMALLINT NOT NULL DEFAULT 1,
    qc_notes          VARCHAR(200)
);
CREATE INDEX idx_receipt_items ON purchase_receipt_items (receipt_id);

-- 16. 盘点单
CREATE TABLE stocktakes (
    id           BIGSERIAL PRIMARY KEY,
    stocktake_no VARCHAR(30) NOT NULL UNIQUE,
    location_id  BIGINT NOT NULL REFERENCES inventory_locations(id),
    type         SMALLINT NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'draft',
    started_by   BIGINT,
    started_at   TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    remarks      VARCHAR(200)
);

-- 17. 盘点明细
CREATE TABLE stocktake_items (
    id               BIGSERIAL PRIMARY KEY,
    stocktake_id     BIGINT NOT NULL REFERENCES stocktakes(id) ON DELETE CASCADE,
    inventory_id     BIGINT NOT NULL REFERENCES inventory(id),
    drug_id          BIGINT NOT NULL,
    batch_no         VARCHAR(50) NOT NULL,
    expiry_date      DATE NOT NULL,
    is_split         BOOLEAN NOT NULL DEFAULT FALSE,
    book_quantity    BIGINT NOT NULL,
    counted_quantity BIGINT,
    difference       BIGINT DEFAULT 0,
    status           VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_stocktake_items ON stocktake_items (stocktake_id);

-- 18. 处方
CREATE TABLE prescriptions (
    id                          BIGSERIAL PRIMARY KEY,
    prescription_no             VARCHAR(30) NOT NULL UNIQUE,
    patient_id                  BIGINT,
    patient_name                VARCHAR(50) NOT NULL,
    patient_gender              VARCHAR(10),
    patient_age                 VARCHAR(10),
    patient_card_no             VARCHAR(50),
    diagnosis                   TEXT,
    department                  VARCHAR(50),
    doctor_name                 VARCHAR(50),
    prescription_type           SMALLINT NOT NULL DEFAULT 0,
    special_control_type        SMALLINT NOT NULL DEFAULT 0,
    source                      VARCHAR(20) NOT NULL DEFAULT 'manual',
    status                      VARCHAR(20) NOT NULL DEFAULT 'pending_review',
    total_amount                BIGINT NOT NULL DEFAULT 0,
    auditor_id                  BIGINT,
    auditor_name                VARCHAR(50),
    dispensing_pharmacist_id    BIGINT,
    dispensing_pharmacist_name  VARCHAR(50),
    checker_id                  BIGINT,
    checker_name                VARCHAR(50),
    reviewed_at                 TIMESTAMPTZ,
    dispensed_at                TIMESTAMPTZ,
    version                     INT NOT NULL DEFAULT 0,
    remarks                     TEXT,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_prescription_status  ON prescriptions (status);
CREATE INDEX idx_prescription_patient ON prescriptions (patient_id);
CREATE INDEX idx_prescription_no      ON prescriptions (prescription_no);

-- 19. 处方明细
CREATE TABLE prescription_items (
    id                BIGSERIAL PRIMARY KEY,
    prescription_id   BIGINT NOT NULL REFERENCES prescriptions(id) ON DELETE CASCADE,
    line_no           INT NOT NULL,
    drug_id           BIGINT NOT NULL REFERENCES drugs(id),
    drug_name         VARCHAR(100) NOT NULL,
    specification     VARCHAR(100),
    manufacturer      VARCHAR(100),
    dosage_form       VARCHAR(30),
    base_unit         VARCHAR(20) NOT NULL,
    split_unit        VARCHAR(20),
    pack_size         INT NOT NULL DEFAULT 1,
    is_split          BOOLEAN NOT NULL DEFAULT FALSE,
    quantity          BIGINT NOT NULL,
    unit_price        BIGINT NOT NULL,
    amount            BIGINT NOT NULL,
    usage_text        VARCHAR(200),
    frequency         VARCHAR(50),
    single_dose       BIGINT,
    total_daily_dose  BIGINT,
    days              INT,
    dispensed_quantity BIGINT NOT NULL DEFAULT 0,
    returned_quantity  BIGINT NOT NULL DEFAULT 0,
    status            VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_presc_items ON prescription_items (prescription_id, drug_id);

-- 20. 发药记录
CREATE TABLE prescription_dispense_records (
    id                BIGSERIAL PRIMARY KEY,
    prescription_id   BIGINT NOT NULL REFERENCES prescriptions(id),
    item_id           BIGINT NOT NULL REFERENCES prescription_items(id),
    inventory_id      BIGINT NOT NULL REFERENCES inventory(id),
    drug_id           BIGINT NOT NULL,
    batch_no          VARCHAR(50) NOT NULL,
    expiry_date       DATE NOT NULL,
    is_split          BOOLEAN NOT NULL DEFAULT FALSE,
    quantity          BIGINT NOT NULL,
    unit_price        BIGINT NOT NULL,
    amount            BIGINT NOT NULL,
    return_quantity   BIGINT NOT NULL DEFAULT 0,
    dispensed_by      BIGINT,
    dispensed_by_name VARCHAR(50),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_dispense_records ON prescription_dispense_records (prescription_id);
CREATE INDEX idx_dispense_inv     ON prescription_dispense_records (inventory_id);
CREATE INDEX idx_dispense_item    ON prescription_dispense_records (item_id);

-- 21. 处方状态流转日志
CREATE TABLE prescription_audit_logs (
    id              BIGSERIAL PRIMARY KEY,
    prescription_id BIGINT NOT NULL REFERENCES prescriptions(id),
    action          VARCHAR(30) NOT NULL,
    from_status     VARCHAR(20),
    to_status       VARCHAR(20),
    operator_id     BIGINT,
    operator_name   VARCHAR(50),
    remarks         VARCHAR(500),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_presc ON prescription_audit_logs (prescription_id);

-- 22. 麻精药品专账
CREATE TABLE special_drug_ledgers (
    id              BIGSERIAL PRIMARY KEY,
    drug_id         BIGINT NOT NULL REFERENCES drugs(id),
    batch_no        VARCHAR(50) NOT NULL,
    prescription_id BIGINT REFERENCES prescriptions(id),
    log_type        VARCHAR(20) NOT NULL,
    quantity        BIGINT NOT NULL,
    patient_name    VARCHAR(50),
    patient_card_no VARCHAR(50),
    operator_id     BIGINT,
    operator_name   VARCHAR(50),
    notes           VARCHAR(200),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_special_ledger ON special_drug_ledgers (drug_id, created_at);

-- 23. 空安瓿回收
CREATE TABLE ampoule_returns (
    id              BIGSERIAL PRIMARY KEY,
    drug_id         BIGINT NOT NULL REFERENCES drugs(id),
    batch_no        VARCHAR(50),
    prescription_id BIGINT REFERENCES prescriptions(id),
    patient_name    VARCHAR(50),
    quantity        INT NOT NULL,
    returned_by     VARCHAR(50),
    verified_by     VARCHAR(50),
    return_date     DATE NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 24. 用药咨询
CREATE TABLE consultations (
    id           BIGSERIAL PRIMARY KEY,
    patient_name VARCHAR(50),
    patient_id   BIGINT,
    drug_id      BIGINT REFERENCES drugs(id),
    question     TEXT NOT NULL,
    answer       TEXT,
    consultant   VARCHAR(50),
    contact      VARCHAR(50),
    consulted_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 25. 不良反应
CREATE TABLE adverse_reactions (
    id            BIGSERIAL PRIMARY KEY,
    patient_name  VARCHAR(50),
    patient_id    BIGINT,
    drug_id       BIGINT REFERENCES drugs(id),
    batch_no      VARCHAR(50),
    reaction_desc TEXT NOT NULL,
    severity      SMALLINT NOT NULL DEFAULT 1,
    outcome       VARCHAR(20),
    reporter      VARCHAR(50),
    report_date   DATE NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 26. 用药指导
CREATE TABLE medication_guidances (
    id              BIGSERIAL PRIMARY KEY,
    prescription_id BIGINT REFERENCES prescriptions(id),
    patient_name    VARCHAR(50),
    drug_id         BIGINT REFERENCES drugs(id),
    content         TEXT NOT NULL,
    pharmacist      VARCHAR(50),
    guided_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- 27. 预警记录
CREATE TABLE stock_alerts (
    id          BIGSERIAL PRIMARY KEY,
    alert_type  VARCHAR(20) NOT NULL,
    drug_id     BIGINT NOT NULL REFERENCES drugs(id),
    location_id BIGINT,
    batch_no    VARCHAR(50),
    expiry_date DATE,
    quantity    BIGINT,
    message     VARCHAR(300),
    status      VARCHAR(20) NOT NULL DEFAULT 'open',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at TIMESTAMPTZ
);
CREATE INDEX idx_alerts_type ON stock_alerts (status, alert_type);
