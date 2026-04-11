-- Rollback: restore incoming_dn_item_id to uuid (matches pre-0015 state).
ALTER TABLE qc_tasks
    ALTER COLUMN incoming_dn_item_id TYPE uuid
    USING NULL;
