-- Rollback 0012: restore default status = Draft

ALTER TABLE po_budget_entries
    ALTER COLUMN status SET DEFAULT 'Draft';
