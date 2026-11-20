-- 0027: Drop the (incoming_dn_item_id, scan_ref) unique constraint.
--
-- scan_ref is set to packing_number, which is the same for every scan of
-- the same DN item. The composite constraint therefore blocks legitimate
-- rescans (e.g. after a QC reject / round-2 scan).
--
-- Idempotency is already enforced by the standalone UNIQUE on idempotency_key
-- (client_event_id per scan event), so the composite constraint is redundant
-- and incorrect.

ALTER TABLE incoming_receiving_scans
DROP CONSTRAINT IF EXISTS incoming_receiving_scans_dn_item_scanref_uq;