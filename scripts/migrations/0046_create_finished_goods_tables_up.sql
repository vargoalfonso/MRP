-- Migration 0046: Create finished_goods and fg_movement_logs tables
--
-- Context (from AGENT/README.md existing design):
--   Legacy table `finished_good` (Sequelize/Node) only has:
--     id(uuid), uniq, quantity, stock_status, warehouse_code
--   This new Go table `finished_goods` replaces it with a richer schema
--   following the same pattern as raw_materials / scrap_stocks.
--
-- Data-source mapping:
--   part_number, part_name, model, uom  ← bom_item (via uniq_code)
--   wo_number                           ← work_orders (auto-filled on create)
--   kanban_standard_qty                 ← kanban_parameters.kanban_qty
--   min_threshold                       ← kanban_parameters.min_stock
--   max_threshold                       ← kanban_parameters.max_stock
--   safety_stock_qty (Target)           ← safety_stock_parameters (or kanban_parameters.max_stock)
--
-- finished_goods       : on-hand FG balance per Uniq (denormalized for fast reads)
-- fg_movement_logs     : append-only ledger — fills the "no central ledger" gap noted in README

-- ---------------------------------------------------------------------------
-- 1. finished_goods
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS finished_goods (
    id                      BIGSERIAL PRIMARY KEY,
    uuid                    UUID        NOT NULL DEFAULT uuid_generate_v4(),

-- Item identity (denormalized from bom_item for fast list reads)
uniq_code VARCHAR(64) NOT NULL UNIQUE,
item_id BIGINT, -- optional FK to bom_item/items (same pattern as raw_materials.item_id)
part_number VARCHAR(128),
part_name VARCHAR(255),
model VARCHAR(128),

-- Traceability
wo_number VARCHAR(128), -- last WO that produced this FG (from work_orders)
warehouse_location VARCHAR(255), -- destination warehouse (from warehouse master)

-- Stock
stock_qty NUMERIC(15, 4) NOT NULL DEFAULT 0,
uom VARCHAR(32), -- pcs | set | kg (from bom_item / uom_parameters)

-- Kanban (denormalized snapshot from kanban_parameters at upsert time)
-- kanban_parameters columns: kanban_qty → kanban_standard_qty
--                            min_stock  → min_threshold
--                            max_stock  → max_threshold
kanban_count INTEGER, -- computed: floor(stock_qty / kanban_standard_qty)
kanban_standard_qty INTEGER, -- mirror of kanban_parameters.kanban_qty
min_threshold NUMERIC(15, 4), -- mirror of kanban_parameters.min_stock  → low_on_stock boundary
max_threshold NUMERIC(15, 4), -- mirror of kanban_parameters.max_stock  → overstock boundary
safety_stock_qty NUMERIC(15, 4), -- target qty shown as "Target" in UI
stock_to_complete_kanban NUMERIC(15, 4), -- computed: max(0, safety_stock_qty - stock_qty)

-- Status flag — recomputed on every stock change
--   stock_qty < min_threshold  → low_on_stock
--   stock_qty > max_threshold  → overstock
--   else                       → normal
status VARCHAR(32) NOT NULL DEFAULT 'normal',

-- Audit
created_by              VARCHAR(255),
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_by              VARCHAR(255),
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_finished_goods_uuid ON finished_goods (uuid);

CREATE INDEX IF NOT EXISTS idx_finished_goods_uniq_code ON finished_goods (uniq_code);

CREATE INDEX IF NOT EXISTS idx_finished_goods_wo_number ON finished_goods (wo_number);

CREATE INDEX IF NOT EXISTS idx_finished_goods_status ON finished_goods (status);

CREATE INDEX IF NOT EXISTS idx_finished_goods_deleted_at ON finished_goods (deleted_at);

-- ---------------------------------------------------------------------------
-- 2. fg_movement_logs — every stock change is recorded here
-- ---------------------------------------------------------------------------


CREATE TABLE IF NOT EXISTS fg_movement_logs (
    id              BIGSERIAL PRIMARY KEY,

    fg_id           BIGINT      NOT NULL REFERENCES finished_goods (id),
    uniq_code       VARCHAR(64) NOT NULL,

-- What changed
movement_type VARCHAR(64) NOT NULL,
-- incoming_production | delivery_scan | manual_add | manual_deduct
-- stock_opname | qc_approved | wo_complete | delete
qty_change NUMERIC(15, 4) NOT NULL, -- positive = increase, negative = decrease
qty_before NUMERIC(15, 4) NOT NULL,
qty_after NUMERIC(15, 4) NOT NULL,

-- References

source_flag     VARCHAR(64),   -- action_ui | manual | stock_opname | delivery
    wo_number       VARCHAR(128),
    dn_number       VARCHAR(128),
    reference_id    VARCHAR(255),  -- any other reference (kanban, packing number, etc.)

    notes           TEXT,

    logged_by       VARCHAR(255),
    logged_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_fg_move_logs_fg_id ON fg_movement_logs (fg_id);

CREATE INDEX IF NOT EXISTS idx_fg_move_logs_uniq_code ON fg_movement_logs (uniq_code);

CREATE INDEX IF NOT EXISTS idx_fg_move_logs_movement_type ON fg_movement_logs (movement_type);

CREATE INDEX IF NOT EXISTS idx_fg_move_logs_logged_at ON fg_movement_logs (logged_at);