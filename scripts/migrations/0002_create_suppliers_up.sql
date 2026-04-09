CREATE TABLE IF NOT EXISTS public.suppliers
(
    id                       BIGINT GENERATED ALWAYS AS IDENTITY,
    uuid                     UUID        NOT NULL,
    supplier_code            VARCHAR(50) NOT NULL,
    supplier_name            VARCHAR(255) NOT NULL,
    contact_person           VARCHAR(255) NOT NULL,
    contact_number           VARCHAR(50) NOT NULL,
    email_address            VARCHAR(255) NOT NULL,
    material_category        VARCHAR(50) NULL,
    full_address             TEXT        NOT NULL,
    city                     VARCHAR(150) NOT NULL,
    province                 VARCHAR(150) NOT NULL,
    country                  VARCHAR(150) NOT NULL,
    tax_id_npwp              VARCHAR(50) NOT NULL,
    bank_name                VARCHAR(150) NOT NULL,
    bank_account_number      VARCHAR(100) NOT NULL,
    bank_account_name        VARCHAR(255) NOT NULL,
    payment_terms            VARCHAR(150) NOT NULL,
    delivery_lead_time_days  INTEGER     NOT NULL DEFAULT 0,
    status                   VARCHAR(20) NOT NULL DEFAULT 'Active',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at               TIMESTAMPTZ NULL,

    CONSTRAINT suppliers_pkey PRIMARY KEY (id),
    CONSTRAINT suppliers_uuid_key UNIQUE (uuid),
    CONSTRAINT suppliers_supplier_code_key UNIQUE (supplier_code),
    CONSTRAINT suppliers_material_category_check CHECK (
        material_category IS NULL OR material_category IN ('Raw Material', 'Indirect Raw Material', 'Subcon')
    ),
    CONSTRAINT suppliers_status_check CHECK (status IN ('Active', 'Inactive'))
);

CREATE INDEX IF NOT EXISTS idx_suppliers_deleted_at ON public.suppliers (deleted_at);
CREATE INDEX IF NOT EXISTS idx_suppliers_name ON public.suppliers (supplier_name);
CREATE INDEX IF NOT EXISTS idx_suppliers_status ON public.suppliers (status);
CREATE INDEX IF NOT EXISTS idx_suppliers_material_category ON public.suppliers (material_category);
