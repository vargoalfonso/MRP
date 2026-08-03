-- Migration 0090: Customer order automation logs
-- Stores rows that failed to enter / failed to be sent (etc.) from the
-- Raigine automation integration for customer orders (PO / DN / SO).

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS customer_order_automation_logs (
    id                   BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    uuid                 UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    document_id          BIGINT REFERENCES customer_order_documents(id) ON DELETE CASCADE,
    document_number      VARCHAR(128),
    row_no               INTEGER NOT NULL DEFAULT 0,
    item_uniq_code       VARCHAR(100),
    part_name            VARCHAR(255),
    description          TEXT,
    qty_active           NUMERIC(15,4) NOT NULL DEFAULT 0,
    failure_reason       TEXT,
    special_instructions TEXT,
    source               VARCHAR(64) NOT NULL DEFAULT 'automation',
    status               VARCHAR(32) NOT NULL DEFAULT 'failed',
    created_at           TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_customer_order_automation_logs_document_id
    ON customer_order_automation_logs (document_id);

CREATE INDEX IF NOT EXISTS idx_customer_order_automation_logs_document_number
    ON customer_order_automation_logs (document_number);

CREATE INDEX IF NOT EXISTS idx_customer_order_automation_logs_item_uniq_code
    ON customer_order_automation_logs (item_uniq_code);

CREATE INDEX IF NOT EXISTS idx_customer_order_automation_logs_created_at
    ON customer_order_automation_logs (created_at DESC);

-- Seed one example failed-automation log row.
INSERT INTO customer_order_automation_logs
    (document_number, row_no, item_uniq_code, part_name, description, qty_active, failure_reason, special_instructions, source, status)
VALUES
    ('PO-TMC-2025-001', 1, 'LV-001', 'Steel Plate', 'Bracket assembly steel plate', 120, 'Qty aktif melebihi kapasitas produksi (gagal terkirim ke automation)', 'Kirim data harian sebelum pukul 09:00 WIB', 'automation', 'failed');
