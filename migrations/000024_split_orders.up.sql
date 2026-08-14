-- 000024_split_orders.up.sql
-- 拆零操作单（docs/13 F4）：操作人/复核人/原因/来源行/结果行，麻精双人复核审计
CREATE TABLE split_orders (
    id              BIGSERIAL PRIMARY KEY,
    split_no        VARCHAR(30)  NOT NULL,
    inventory_id    BIGINT       NOT NULL,
    drug_id         BIGINT       NOT NULL,
    location_id     BIGINT       NOT NULL,
    batch_no        VARCHAR(50)  NOT NULL,
    expiry_date     DATE,
    boxes           BIGINT       NOT NULL,
    units           BIGINT       NOT NULL,
    damaged         BIGINT       NOT NULL DEFAULT 0,
    split_unit_cost BIGINT       NOT NULL DEFAULT 0, -- 拆零行进价（分/拆零单位）
    operator_id     BIGINT,
    operator_name   VARCHAR(50),
    reviewer_id     BIGINT,                            -- 麻精双人复核人（普通拆零可空）
    reviewer_name   VARCHAR(50),
    remarks         VARCHAR(200),
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    CONSTRAINT uq_split_orders_no UNIQUE (split_no)
);
CREATE INDEX idx_split_orders_drug ON split_orders (drug_id, created_at);
