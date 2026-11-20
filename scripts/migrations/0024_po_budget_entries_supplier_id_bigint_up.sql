-- Change po_budget_entries.supplier_id from varchar/uuid to bigint (legacy supplier.supplier_id).
-- Existing rows with non-numeric values are cleared to NULL before the cast.
ALTER TABLE po_budget_entries
    ALTER COLUMN supplier_id DROP DEFAULT,
    ALTER COLUMN supplier_id TYPE bigint
        USING CASE
            WHEN supplier_id::text ~ '^\d+$' THEN supplier_id::text::bigint
            ELSE NULL
        END;


ALTER TABLE po_budget_entries
    ALTER COLUMN supplier_id DROP DEFAULT,
    ALTER COLUMN supplier_id TYPE bigint
        USING CASE
            WHEN supplier_id::text ~ '^\d+$' THEN supplier_id::text::bigint
            ELSE NULL
        END;