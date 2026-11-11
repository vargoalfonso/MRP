-- Migration 0057: Create unified QC dashboard tables and scrap linkage

CREATE TABLE IF NOT EXISTS qc_logs (
    id BIGSERIAL PRIMARY KEY,
    uuid UUID NOT NULL DEFAULT uuid_generate_v4() UNIQUE,
    qc_task_id BIGINT REFERENCES qc_tasks(id) ON DELETE SET NULL,
    wo_id BIGINT REFERENCES work_orders(id) ON DELETE SET NULL,
    wo_item_id BIGINT REFERENCES work_order_items(id) ON DELETE SET NULL,
    dn_item_id BIGINT REFERENCES delivery_note_items(id) ON DELETE SET NULL,
    uniq_code VARCHAR(64) NOT NULL,
    qc_round SMALLINT NOT NULL DEFAULT 1,
    qty_checked NUMERIC(15,4) NOT NULL DEFAULT 0,
    qty_pass NUMERIC(15,4) NOT NULL DEFAULT 0,
    qty_defect NUMERIC(15,4) NOT NULL DEFAULT 0,
    qty_scrap NUMERIC(15,4) NOT NULL DEFAULT 0,
    status VARCHAR(32) NOT NULL,
    defect_source VARCHAR(32),
    checked_by VARCHAR(255),
    checked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_qc_logs_checked_at ON qc_logs (checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_qc_logs_wo_id_checked_at ON qc_logs (wo_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_qc_logs_dn_item_id ON qc_logs (dn_item_id);
CREATE INDEX IF NOT EXISTS idx_qc_logs_uniq_checked_at ON qc_logs (uniq_code, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_qc_logs_defect_source ON qc_logs (defect_source, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_qc_logs_qc_task_id ON qc_logs (qc_task_id);

CREATE TABLE IF NOT EXISTS qc_defect_items (
    id BIGSERIAL PRIMARY KEY,
    qc_log_id BIGINT NOT NULL REFERENCES qc_logs(id) ON DELETE CASCADE,
    qc_task_id BIGINT REFERENCES qc_tasks(id) ON DELETE SET NULL,
    wo_id BIGINT REFERENCES work_orders(id) ON DELETE SET NULL,
    wo_item_id BIGINT REFERENCES work_order_items(id) ON DELETE SET NULL,
    dn_item_id BIGINT REFERENCES delivery_note_items(id) ON DELETE SET NULL,
    uniq_code VARCHAR(64) NOT NULL,
    defect_source VARCHAR(32) NOT NULL,
    defect_reason_code VARCHAR(64),
    defect_reason_text VARCHAR(255),
    qty_defect NUMERIC(15,4) NOT NULL DEFAULT 0,
    qty_scrap NUMERIC(15,4) NOT NULL DEFAULT 0,
    is_repairable BOOLEAN NOT NULL DEFAULT FALSE,
    machine_id UUID,
    process_name VARCHAR(64),
    reported_by VARCHAR(255),
    reported_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_qc_defect_items_reported_at ON qc_defect_items (reported_at DESC);
CREATE INDEX IF NOT EXISTS idx_qc_defect_items_source ON qc_defect_items (defect_source, reported_at DESC);
CREATE INDEX IF NOT EXISTS idx_qc_defect_items_uniq ON qc_defect_items (uniq_code, reported_at DESC);
CREATE INDEX IF NOT EXISTS idx_qc_defect_items_reason_code ON qc_defect_items (defect_reason_code);
CREATE INDEX IF NOT EXISTS idx_qc_defect_items_dn_item ON qc_defect_items (dn_item_id);
CREATE INDEX IF NOT EXISTS idx_qc_defect_items_qc_log_id ON qc_defect_items (qc_log_id);

ALTER TABLE scrap_stocks
    ADD COLUMN IF NOT EXISTS source_qc_log_id BIGINT REFERENCES qc_logs(id) ON DELETE SET NULL,
    ADD COLUMN IF NOT EXISTS source_defect_id BIGINT REFERENCES qc_defect_items(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_scrap_stocks_source_qc_log ON scrap_stocks (source_qc_log_id);
CREATE INDEX IF NOT EXISTS idx_scrap_stocks_source_defect ON scrap_stocks (source_defect_id);
