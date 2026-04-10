-- Migration 0013: PO Budget entry history logs

CREATE TABLE IF NOT EXISTS po_budget_entry_logs (
    id         bigserial PRIMARY KEY,
    entry_id   bigint      NOT NULL REFERENCES po_budget_entries(id) ON DELETE CASCADE,
    action     varchar(32) NOT NULL CHECK (action IN ('Created','Submitted','Updated','Approved','Rejected')),
    username   varchar(255),
    notes      text,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_po_budget_entry_logs_entry_id ON po_budget_entry_logs (entry_id);
CREATE INDEX IF NOT EXISTS idx_po_budget_entry_logs_created  ON po_budget_entry_logs (created_at DESC);
