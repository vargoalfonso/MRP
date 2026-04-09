-- =============================================================================
-- Migration: 0003_create_bom_tables_up.sql
-- Module   : Bill of Material (BOM) + Routing + Material Spec
-- BRD      : Products > Bill of Material
-- =============================================================================

-- =============================================================================
-- EXTENSION
-- =============================================================================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- MASTER: uom_parameters  (unit of measure)
-- =============================================================================
CREATE TABLE IF NOT EXISTS uom_parameters (
    id          BIGSERIAL    PRIMARY KEY,
    code        VARCHAR(20)  NOT NULL UNIQUE,
    name        VARCHAR(100) NOT NULL,
    category    VARCHAR(50)  NULL,
    status      VARCHAR(20)  NOT NULL DEFAULT 'Active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- MASTER: process_parameters  (stamping, bending, assy, etc.)
-- =============================================================================
CREATE TABLE IF NOT EXISTS process_parameters (
    id            BIGSERIAL    PRIMARY KEY,
    process_code  VARCHAR(50)  NOT NULL UNIQUE,
    process_name  VARCHAR(150) NOT NULL,
    category      VARCHAR(50)  NULL,
    sequence      INT          NOT NULL DEFAULT 0,
    status        VARCHAR(20)  NOT NULL DEFAULT 'Active',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- MASTER: master_machines
-- =============================================================================
CREATE TABLE IF NOT EXISTS master_machines (
    id               BIGSERIAL    PRIMARY KEY,
    machine_number   VARCHAR(50)  NOT NULL UNIQUE,
    machine_name     VARCHAR(150) NOT NULL,
    production_line  VARCHAR(100) NULL,
    process_id       BIGINT       NULL REFERENCES process_parameters(id),
    machine_capacity INT          NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'Active',
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- NOTE: suppliers table already exists in DB (uuid PK, managed separately).
--       item_material_specs.supplier_id references it via UUID FK.

-- =============================================================================
-- CORE: items  (parent & child parts — every uniq_code lives here)
-- =============================================================================
CREATE TABLE IF NOT EXISTS items (
    id                       BIGSERIAL    PRIMARY KEY,
    uuid                     UUID         NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    uniq_code                VARCHAR(64)  NOT NULL UNIQUE,   -- e.g. LV7, LV8, MB6
    part_number              VARCHAR(128) NULL,
    part_name                VARCHAR(255) NOT NULL,
    model                    VARCHAR(128) NULL,
    material_type            VARCHAR(64)  NULL,
    -- Make / Buy / Make-or-Buy
    sourcing_type            VARCHAR(16)  NULL CHECK (sourcing_type IN ('Make','Buy','Make-or-Buy')),
    uom_id                   BIGINT       NOT NULL REFERENCES uom_parameters(id),
    standard_weight_kg       NUMERIC(18,6) NULL,
    requires_weight_tracking BOOLEAN      NOT NULL DEFAULT FALSE,
    current_revision         VARCHAR(32)  NULL,
    status                   VARCHAR(20)  NOT NULL DEFAULT 'Active' CHECK (status IN ('Active','Inactive','Obsolete')),
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at               TIMESTAMPTZ  NULL
);

CREATE INDEX IF NOT EXISTS idx_items_uniq_code ON items(uniq_code) WHERE deleted_at IS NULL;

-- =============================================================================
-- CORE: item_revisions  (drawing / part-number version control)
-- =============================================================================
CREATE TABLE IF NOT EXISTS item_revisions (
    id              BIGSERIAL    PRIMARY KEY,
    uuid            UUID         NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    item_id         BIGINT       NOT NULL REFERENCES items(id),
    revision        VARCHAR(32)  NOT NULL,          -- e.g. "A", "B", "Rev-2"
    -- Draft / Released / Obsolete
    status          VARCHAR(20)  NOT NULL DEFAULT 'Draft' CHECK (status IN ('Draft','Released','Obsolete')),
    effective_from  DATE         NULL,
    effective_to    DATE         NULL,
    change_note     TEXT         NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (item_id, revision)
);

CREATE INDEX IF NOT EXISTS idx_item_revisions_item_id ON item_revisions(item_id);

-- =============================================================================
-- CORE: item_material_specs  (1:1 per revision)
-- Material Code/Grade, Form, dimensions, supplier, cycle time, dandori
-- =============================================================================
CREATE TABLE IF NOT EXISTS item_material_specs (
    id                BIGSERIAL    PRIMARY KEY,
    item_revision_id  BIGINT       NOT NULL UNIQUE REFERENCES item_revisions(id),
    material_grade    VARCHAR(64)  NULL,   -- e.g. STKM550
    -- Plate / Coil / Pipe / Rod / Wire
    form              VARCHAR(32)  NULL CHECK (form IN ('Plate','Coil','Pipe','Rod','Wire','Other')),
    width_mm          NUMERIC(18,4) NULL,
    diameter_mm       NUMERIC(18,4) NULL,
    thickness_mm      NUMERIC(18,4) NULL,
    length_mm         NUMERIC(18,4) NULL,
    weight_kg         NUMERIC(18,6) NULL,   -- for wire
    supplier_id       UUID         NULL REFERENCES suppliers(id),
    cycle_time_sec    NUMERIC(18,4) NULL,   -- sec/pc
    setup_time_min    NUMERIC(18,4) NULL,   -- dandori in min
    notes             TEXT         NULL,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- CORE: item_assets  (pictures for parent & child parts)
-- =============================================================================
CREATE TABLE IF NOT EXISTS item_assets (
    id                BIGSERIAL    PRIMARY KEY,
    uuid              UUID         NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    item_id           BIGINT       NOT NULL REFERENCES items(id),
    item_revision_id  BIGINT       NULL REFERENCES item_revisions(id),
    -- drawing / photo / 3d-model
    asset_type        VARCHAR(32)  NOT NULL CHECK (asset_type IN ('drawing','photo','3d-model','other')),
    revision          VARCHAR(32)  NULL,
    title             VARCHAR(255) NULL,
    file_url          TEXT         NOT NULL,
    status            VARCHAR(20)  NOT NULL DEFAULT 'Active',
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_item_assets_item_id ON item_assets(item_id);

-- =============================================================================
-- CORE: routing_headers  (one routing per item × version)
-- Carries op-seq ordering that enforces poka-yoke in Action UI
-- =============================================================================
CREATE TABLE IF NOT EXISTS routing_headers (
    id                BIGSERIAL    PRIMARY KEY,
    uuid              UUID         NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    item_id           BIGINT       NOT NULL REFERENCES items(id),
    item_revision_id  BIGINT       NULL REFERENCES item_revisions(id),
    version           INT          NOT NULL DEFAULT 1,
    -- Draft / Released / Obsolete
    status            VARCHAR(20)  NOT NULL DEFAULT 'Draft' CHECK (status IN ('Draft','Released','Obsolete')),
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (item_id, version)
);

CREATE INDEX IF NOT EXISTS idx_routing_headers_item_id ON routing_headers(item_id);

-- =============================================================================
-- CORE: routing_operations  (op-seq per routing header)
-- Cycle time + dandori used by production planning capacity calc
-- =============================================================================
CREATE TABLE IF NOT EXISTS routing_operations (
    id                 BIGSERIAL     PRIMARY KEY,
    routing_header_id  BIGINT        NOT NULL REFERENCES routing_headers(id) ON DELETE CASCADE,
    op_seq             INT           NOT NULL,          -- 10, 20, 30 …
    process_id         BIGINT        NOT NULL REFERENCES process_parameters(id),
    machine_id         BIGINT        NULL REFERENCES master_machines(id),
    cycle_time_sec     NUMERIC(18,4) NULL,              -- sec/pc
    setup_time_min     NUMERIC(18,4) NULL,              -- dandori min
    machine_stroke     VARCHAR(100)           NULL,              -- strokes / min
    notes              TEXT          NULL,
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    UNIQUE (routing_header_id, op_seq)
);

CREATE INDEX IF NOT EXISTS idx_routing_ops_header_id ON routing_operations(routing_header_id);

-- =============================================================================
-- CORE: bom_item  (BOM header — ties a root item to its version)
-- =============================================================================
CREATE TABLE IF NOT EXISTS bom_item (
    id                    BIGSERIAL    PRIMARY KEY,
    uuid                  UUID         NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
    item_id               BIGINT       NOT NULL REFERENCES items(id),
    root_item_revision_id BIGINT       NULL REFERENCES item_revisions(id),
    version               INT          NOT NULL DEFAULT 1,
    -- Draft / Released / Obsolete
    status                VARCHAR(20)  NOT NULL DEFAULT 'Draft' CHECK (status IN ('Draft','Released','Obsolete')),
    effective_from        DATE         NULL,
    effective_to          DATE         NULL,
    description           TEXT         NULL,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (item_id, version)
);

CREATE INDEX IF NOT EXISTS idx_bom_item_item_id ON bom_item(item_id);

-- =============================================================================
-- CORE: bom_lines  (parent → child explosion, multilevel up to 4 levels)
-- qty_per_uniq (QPU): how many child parts needed per parent assembly
-- =============================================================================
CREATE TABLE IF NOT EXISTS bom_lines (
    id                      BIGSERIAL     PRIMARY KEY,
    bom_item_id             BIGINT        NOT NULL REFERENCES bom_item(id) ON DELETE CASCADE,
    parent_item_id          BIGINT        NOT NULL REFERENCES items(id),
    child_item_id           BIGINT        NOT NULL REFERENCES items(id),
    level                   SMALLINT      NOT NULL DEFAULT 1 CHECK (level BETWEEN 1 AND 4),
    line_no                 INT           NULL,
    qty_per_uniq            NUMERIC(18,6) NOT NULL DEFAULT 1,   -- QPU
    uom_id                  BIGINT        NULL REFERENCES uom_parameters(id),
    child_item_revision_id  BIGINT        NULL REFERENCES item_revisions(id),
    scrap_factor            NUMERIC(9,6)  NOT NULL DEFAULT 0,
    is_phantom              BOOLEAN       NOT NULL DEFAULT FALSE,
    notes                   TEXT          NULL,
    created_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bom_lines_bom_item_id    ON bom_lines(bom_item_id);
CREATE INDEX IF NOT EXISTS idx_bom_lines_parent_item_id ON bom_lines(parent_item_id);
CREATE INDEX IF NOT EXISTS idx_bom_lines_child_item_id  ON bom_lines(child_item_id);

-- =============================================================================
-- CORE: bom_approvals  (manager sign-off on a BOM version)
-- =============================================================================
CREATE TABLE IF NOT EXISTS bom_approvals (
    id            BIGSERIAL    PRIMARY KEY,
    bom_item_id   BIGINT       NOT NULL REFERENCES bom_item(id) ON DELETE CASCADE,
    approved_by   UUID         NULL,   -- users.uuid
    -- pending / approved / rejected
    status        VARCHAR(20)  NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','approved','rejected')),
    notes         TEXT         NULL,
    approved_at   TIMESTAMPTZ  NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_bom_approvals_bom_item_id ON bom_approvals(bom_item_id);

-- =============================================================================
-- OPTIONAL: bom_closure  (pre-computed ancestor/descendant for fast BOM explosion)
-- Populated by trigger or application-level backfill.
-- =============================================================================
CREATE TABLE IF NOT EXISTS bom_closure (
    ancestor_item_id    BIGINT   NOT NULL REFERENCES items(id),
    descendant_item_id  BIGINT   NOT NULL REFERENCES items(id),
    depth               INT      NOT NULL,
    bom_item_id         BIGINT   NULL REFERENCES bom_item(id),
    PRIMARY KEY (ancestor_item_id, descendant_item_id, bom_item_id)
);
