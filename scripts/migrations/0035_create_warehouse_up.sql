CREATE TABLE IF NOT EXISTS public.warehouse (
    id              BIGINT GENERATED ALWAYS AS IDENTITY,
    uuid            UUID         NOT NULL,
    warehouse_name  VARCHAR(255) NOT NULL,
    type_warehouse  VARCHAR(50)  NOT NULL,
    plant_id        VARCHAR(100) NOT NULL,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ  NULL,

    CONSTRAINT warehouse_pkey PRIMARY KEY (id),
    CONSTRAINT warehouse_uuid_key UNIQUE (uuid),
    CONSTRAINT warehouse_name_plant_key UNIQUE (warehouse_name, plant_id),
    CONSTRAINT warehouse_type_check CHECK (type_warehouse IN ('raw_material', 'wip', 'finished_goods', 'subcon', 'general'))
);

CREATE INDEX IF NOT EXISTS idx_warehouse_deleted_at ON public.warehouse (deleted_at);
CREATE INDEX IF NOT EXISTS idx_warehouse_type ON public.warehouse (type_warehouse);
CREATE INDEX IF NOT EXISTS idx_warehouse_plant_id ON public.warehouse (plant_id);
