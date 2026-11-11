-- Migration 0012: PO Budget default status = Pending

ALTER TABLE po_budget_entries
    ALTER COLUMN status SET DEFAULT 'Pending';
