-- 000027_requisition_orders.up.sql
-- 耗材领用/补发登记单（docs/18：药房护士核心职能）
-- 多明细一次领用，FEFO 扣减并留痕；purpose: clinical=临床领用 / supplement=补发 / other
CREATE TABLE requisition_orders (
    id              BIGSERIAL PRIMARY KEY,
    requisition_no  VARCHAR(30) NOT NULL,
    location_id     BIGINT      NOT NULL,
    purpose         VARCHAR(20) NOT NULL DEFAULT 'supplement',
    reason          VARCHAR(200),
    operator_id     BIGINT,
    operator_name   VARCHAR(50),
    status          VARCHAR(20) NOT NULL DEFAULT 'completed',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_requisition_orders_no UNIQUE (requisition_no)
);
CREATE INDEX idx_requisition_orders_loc ON requisition_orders (location_id, created_at);

CREATE TABLE requisition_order_items (
    id                   BIGSERIAL PRIMARY KEY,
    requisition_order_id BIGINT NOT NULL REFERENCES requisition_orders(id) ON DELETE CASCADE,
    drug_id              BIGINT NOT NULL,
    batch_no             VARCHAR(50),
    is_split             BOOLEAN NOT NULL DEFAULT FALSE,
    quantity             BIGINT NOT NULL,
    unit_price           BIGINT NOT NULL DEFAULT 0,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_requisition_items_order ON requisition_order_items (requisition_order_id);
CREATE INDEX idx_requisition_items_drug ON requisition_order_items (drug_id);
