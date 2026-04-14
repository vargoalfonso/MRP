CREATE TABLE IF NOT EXISTS public.supplier_item
(
    id               BIGINT GENERATED ALWAYS AS IDENTITY,
    uuid             UUID           NOT NULL,
    supplier_uuid    UUID           NOT NULL,
    supplier_name    VARCHAR(255)   NOT NULL,
    sebango_code     VARCHAR(100)   NOT NULL,
    uniq_code        VARCHAR(100)   NOT NULL,
    type             VARCHAR(32)    NOT NULL,
    description      TEXT           NULL,
    quantity         BIGINT         NOT NULL DEFAULT 0,
    uom              VARCHAR(32)    NULL,
    weight           NUMERIC(15,4)  NULL,
    pcs_per_kanban   BIGINT         NULL,
    customer_cycle   VARCHAR(100)   NULL,
    status           VARCHAR(20)    NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ    NOT NULL DEFAULT NOW(),
    deleted_at       TIMESTAMPTZ    NULL,

    CONSTRAINT supplier_item_pkey PRIMARY KEY (id),
    CONSTRAINT supplier_item_uuid_key UNIQUE (uuid),
    CONSTRAINT supplier_item_supplier_uniq_key UNIQUE (supplier_uuid, uniq_code),
    CONSTRAINT supplier_item_type_check CHECK (type IN ('raw_material', 'indirect', 'subcon')),
    CONSTRAINT supplier_item_status_check CHECK (status IN ('active', 'inactive'))
);

CREATE INDEX IF NOT EXISTS idx_supplier_item_deleted_at ON public.supplier_item (deleted_at);
CREATE INDEX IF NOT EXISTS idx_supplier_item_supplier_uuid ON public.supplier_item (supplier_uuid);
CREATE INDEX IF NOT EXISTS idx_supplier_item_type ON public.supplier_item (type);
CREATE INDEX IF NOT EXISTS idx_supplier_item_uniq_code ON public.supplier_item (uniq_code);
