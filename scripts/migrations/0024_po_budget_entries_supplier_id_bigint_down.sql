ALTER TABLE po_budget_entries
    ALTER COLUMN supplier_id TYPE varchar(36)
        USING supplier_id::varchar;
