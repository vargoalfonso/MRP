-- Ensure product_returns exists (it was created at runtime, not via migration).
CREATE TABLE IF NOT EXISTS product_returns (
    id              BIGSERIAL PRIMARY KEY,
    uniq            VARCHAR(100) NOT NULL,
    dn_number       VARCHAR(100) NOT NULL,
    quantity_scrap  INT DEFAULT 0,
    quantity_rework INT DEFAULT 0,
    status          VARCHAR(50) DEFAULT 'PENDING',
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now()
);

-- BRD fields for Product Return.
ALTER TABLE product_returns ADD COLUMN IF NOT EXISTS weight NUMERIC(18,6) DEFAULT 0;
ALTER TABLE product_returns ADD COLUMN IF NOT EXISTS uom VARCHAR(50);
ALTER TABLE product_returns ADD COLUMN IF NOT EXISTS date_received DATE;
ALTER TABLE product_returns ADD COLUMN IF NOT EXISTS scrap_type VARCHAR(50) DEFAULT 'Product Return';

CREATE INDEX IF NOT EXISTS idx_product_returns_uniq ON product_returns (uniq);
