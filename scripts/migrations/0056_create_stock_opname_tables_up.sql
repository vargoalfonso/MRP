CREATE TABLE IF NOT EXISTS stock_opname_sessions (
    id                  BIGSERIAL PRIMARY KEY,
    uuid                UUID        NOT NULL DEFAULT uuid_generate_v4(),
    session_number      VARCHAR(64) NOT NULL,
    inventory_type      VARCHAR(16) NOT NULL,
    method              VARCHAR(16) NOT NULL DEFAULT 'manual',
    period_month        INT         NOT NULL CHECK (period_month BETWEEN 1 AND 12),
    period_year         INT         NOT NULL,
    warehouse_location  VARCHAR(255),
    schedule_date       DATE,
    counted_date        DATE,
    remarks             TEXT,
    total_entries       INT           NOT NULL DEFAULT 0,
    total_variance_qty  NUMERIC(15,4) NOT NULL DEFAULT 0,
    status              VARCHAR(32) NOT NULL DEFAULT 'draft',
    submitted_by        VARCHAR(255),
    submitted_at        TIMESTAMPTZ,
    approver            VARCHAR(255),
    approved_by         VARCHAR(255),
    approved_at         TIMESTAMPTZ,
    approval_remarks    TEXT,
    created_by          VARCHAR(255),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          VARCHAR(255),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_so_sessions_uuid ON stock_opname_sessions (uuid);
CREATE UNIQUE INDEX IF NOT EXISTS idx_so_sessions_session_number ON stock_opname_sessions (session_number);
CREATE INDEX IF NOT EXISTS idx_so_sessions_type_period ON stock_opname_sessions (inventory_type, period_year, period_month);
CREATE INDEX IF NOT EXISTS idx_so_sessions_status ON stock_opname_sessions (status);
CREATE INDEX IF NOT EXISTS idx_so_sessions_deleted_at ON stock_opname_sessions (deleted_at);

CREATE TABLE IF NOT EXISTS stock_opname_entries (
    id                  BIGSERIAL PRIMARY KEY,
    uuid                UUID        NOT NULL DEFAULT uuid_generate_v4(),
    session_id          BIGINT      NOT NULL REFERENCES stock_opname_sessions (id) ON DELETE CASCADE,
    uniq_code           VARCHAR(64) NOT NULL,
    entity_id           BIGINT,
    part_number         VARCHAR(128),
    part_name           VARCHAR(255),
    uom                 VARCHAR(32),
    system_qty_snapshot NUMERIC(15,4) NOT NULL,
    counted_qty         NUMERIC(15,4) NOT NULL,
    variance_qty        NUMERIC(15,4) GENERATED ALWAYS AS (counted_qty - system_qty_snapshot) STORED,
    variance_pct        NUMERIC(15,4),
    weight_kg           NUMERIC(15,4),
    cycle_pengiriman    VARCHAR(64),
    user_counter        VARCHAR(255),
    remarks             TEXT,
    status              VARCHAR(16) NOT NULL DEFAULT 'pending',
    approved_by         VARCHAR(255),
    approved_at         TIMESTAMPTZ,
    reject_reason       TEXT,
    created_by          VARCHAR(255),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by          VARCHAR(255),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_so_entries_uuid ON stock_opname_entries (uuid);
CREATE INDEX IF NOT EXISTS idx_so_entries_session_id ON stock_opname_entries (session_id);
CREATE INDEX IF NOT EXISTS idx_so_entries_uniq_code ON stock_opname_entries (uniq_code);
CREATE INDEX IF NOT EXISTS idx_so_entries_status ON stock_opname_entries (status);

CREATE TABLE IF NOT EXISTS stock_opname_audit_logs (
    id                BIGSERIAL PRIMARY KEY,
    uuid              UUID         NOT NULL DEFAULT uuid_generate_v4(),
    session_id        BIGINT       NOT NULL REFERENCES stock_opname_sessions (id) ON DELETE CASCADE,
    entry_id          BIGINT,
    inventory_type    VARCHAR(16)  NOT NULL,
    action            VARCHAR(64)  NOT NULL,
    entity_type       VARCHAR(32)  NOT NULL,
    actor             VARCHAR(255) NOT NULL,
    remarks           TEXT,
    metadata          JSONB,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_so_audit_uuid ON stock_opname_audit_logs (uuid);
CREATE INDEX IF NOT EXISTS idx_so_audit_session_id ON stock_opname_audit_logs (session_id);
CREATE INDEX IF NOT EXISTS idx_so_audit_entry_id ON stock_opname_audit_logs (entry_id);
CREATE INDEX IF NOT EXISTS idx_so_audit_inventory_type ON stock_opname_audit_logs (inventory_type);
CREATE INDEX IF NOT EXISTS idx_so_audit_action ON stock_opname_audit_logs (action);
CREATE INDEX IF NOT EXISTS idx_so_audit_created_at ON stock_opname_audit_logs (created_at);
