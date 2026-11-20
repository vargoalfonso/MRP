ALTER TABLE purchase_order_items
    ADD COLUMN IF NOT EXISTS packing_number varchar(64);
