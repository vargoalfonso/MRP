-- 0108: stok WIP per item (hasil WO New untuk child part / item di tengah BOM)
--       + lock stok komponen untuk WO Assembly.
--
-- wip_inventories          : saldo stok WIP per UNIQ (dipakai WO Assembly sebagai komponen)
-- wip_inventory_movements  : ledger append-only perubahan stok WIP
-- assembly_stock_locks     : stok komponen yang "dikunci" untuk WO Assembly agar tidak
--                            dipakai ganda. Stok baru berkurang saat Scan Out proses assembly.
--                            tersedia = stok - SUM(qty_locked - qty_consumed) WHERE status='active'

CREATE TABLE IF NOT EXISTS wip_inventories (
    id          BIGSERIAL     PRIMARY KEY,
    uniq_code   VARCHAR(64)   NOT NULL UNIQUE,
    item_id     BIGINT        NULL,
    part_number VARCHAR(128)  NULL,
    part_name   VARCHAR(255)  NULL,
    model       VARCHAR(128)  NULL,
    uom         VARCHAR(32)   NULL,
    stock_qty   NUMERIC(15,4) NOT NULL DEFAULT 0 CHECK (stock_qty >= 0),
    wo_number   VARCHAR(128)  NULL,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS wip_inventory_movements (
    id                BIGSERIAL     PRIMARY KEY,
    wip_inventory_id  BIGINT        NOT NULL REFERENCES wip_inventories(id) ON DELETE CASCADE,
    uniq_code         VARCHAR(64)   NOT NULL,
    movement_type     VARCHAR(32)   NOT NULL,
    qty_change        NUMERIC(15,4) NOT NULL,
    qty_before        NUMERIC(15,4) NOT NULL,
    qty_after         NUMERIC(15,4) NOT NULL,
    wo_number         VARCHAR(128)  NULL,
    kanban_number     VARCHAR(150)  NULL,
    notes             TEXT          NULL,
    created_by        VARCHAR(255)  NULL,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_wip_inv_mov_uniq ON wip_inventory_movements (uniq_code, created_at DESC);

CREATE TABLE IF NOT EXISTS assembly_stock_locks (
    id                  BIGSERIAL     PRIMARY KEY,
    wo_id               BIGINT        NOT NULL,
    wo_number           VARCHAR(64)   NOT NULL,
    parent_uniq_code    VARCHAR(64)   NOT NULL,
    component_uniq_code VARCHAR(64)   NOT NULL,
    source              VARCHAR(24)   NOT NULL CHECK (source IN ('wip','raw_material','indirect_raw_material')),
    qty_per_uniq        NUMERIC(18,6) NOT NULL,
    qty_locked          NUMERIC(15,4) NOT NULL CHECK (qty_locked >= 0),
    qty_consumed        NUMERIC(15,4) NOT NULL DEFAULT 0 CHECK (qty_consumed >= 0),
    status              VARCHAR(16)   NOT NULL DEFAULT 'active' CHECK (status IN ('active','consumed','released')),
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_asm_locks_wo ON assembly_stock_locks (wo_id, parent_uniq_code) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_asm_locks_component ON assembly_stock_locks (component_uniq_code, source) WHERE status = 'active';
