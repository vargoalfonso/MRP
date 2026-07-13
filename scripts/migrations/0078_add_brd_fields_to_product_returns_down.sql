DROP INDEX IF EXISTS idx_product_returns_uniq;
ALTER TABLE product_returns DROP COLUMN IF EXISTS scrap_type;
ALTER TABLE product_returns DROP COLUMN IF EXISTS date_received;
ALTER TABLE product_returns DROP COLUMN IF EXISTS uom;
ALTER TABLE product_returns DROP COLUMN IF EXISTS weight;
