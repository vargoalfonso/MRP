-- Migration 0010 rollback
ALTER TABLE po_budget_entries
    DROP COLUMN IF EXISTS prl_id,
    DROP COLUMN IF EXISTS prl_item_id,
    DROP COLUMN IF EXISTS budget_qty,
    DROP COLUMN IF EXISTS budget_subtype;

DROP TABLE IF EXISTS prl_forecast_items;
DROP TABLE IF EXISTS prl_forecasts;
