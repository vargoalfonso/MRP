-- Drop redundant counter columns from purchase_orders.
-- dn_created, dn_incoming, total_incoming are derivable from delivery_notes
-- and delivery_note_items; keeping them caused stale data and confusion.
ALTER TABLE purchase_orders
    DROP COLUMN IF EXISTS total_incoming,
    DROP COLUMN IF EXISTS dn_created,
    DROP COLUMN IF EXISTS dn_incoming;
