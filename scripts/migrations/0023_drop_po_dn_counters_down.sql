ALTER TABLE purchase_orders
    ADD COLUMN IF NOT EXISTS total_incoming int NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS dn_created     int NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS dn_incoming    int NOT NULL DEFAULT 0;
