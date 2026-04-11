-- Restore the (incoming_dn_item_id, scan_ref) unique constraint if needed.
ALTER TABLE incoming_receiving_scans
ADD CONSTRAINT incoming_receiving_scans_dn_item_scanref_uq UNIQUE (incoming_dn_item_id, scan_ref);