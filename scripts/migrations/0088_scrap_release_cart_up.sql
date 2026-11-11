-- Migration 0088: multi-item scrap release (cart) + weight(kg)/price-per-kg
-- price_per_kg : sale price per kilogram (all uniq weighed the same)
-- items_json   : JSON snapshot of cart lines [{scrap_stock_id, uniq, part_name, release_qty}]
ALTER TABLE scrap_releases ADD COLUMN IF NOT EXISTS price_per_kg NUMERIC(15, 4);
ALTER TABLE scrap_releases ADD COLUMN IF NOT EXISTS items_json TEXT;
