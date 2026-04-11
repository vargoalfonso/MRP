-- +migrate Down

DROP TABLE IF EXISTS inventory_movement_logs;

DROP TABLE IF EXISTS subcon_inventories;

DROP TABLE IF EXISTS indirect_raw_materials;

DROP TABLE IF EXISTS raw_materials;