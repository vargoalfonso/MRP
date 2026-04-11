CREATE TABLE IF NOT EXISTS public.customers
(
    id                        BIGINT GENERATED ALWAYS AS IDENTITY,
    uuid                      UUID         NOT NULL,
    customer_id               VARCHAR(32)  NOT NULL,
    customer_name             VARCHAR(255) NOT NULL,
    phone_number              VARCHAR(50)  NOT NULL,
    shipping_address          TEXT         NOT NULL,
    billing_address           TEXT         NOT NULL,
    billing_same_as_shipping  BOOLEAN      NOT NULL DEFAULT FALSE,
    bank_account              VARCHAR(150) NULL,
    bank_account_number       VARCHAR(100) NULL,
    created_at                TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at                TIMESTAMPTZ  NULL,

    CONSTRAINT customers_pkey PRIMARY KEY (id),
    CONSTRAINT customers_uuid_key UNIQUE (uuid),
    CONSTRAINT customers_customer_id_key UNIQUE (customer_id)
);

CREATE INDEX IF NOT EXISTS idx_customers_deleted_at ON public.customers (deleted_at);
CREATE INDEX IF NOT EXISTS idx_customers_customer_name ON public.customers (customer_name);
CREATE INDEX IF NOT EXISTS idx_customers_customer_id ON public.customers (customer_id);

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'suppliers'
    )
    AND NOT EXISTS (
        SELECT 1
        FROM information_schema.columns
        WHERE table_schema = 'public' AND table_name = 'suppliers' AND column_name = 'uuid'
    )
    AND NOT EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'suppliers_legacy_0006'
    ) THEN
        ALTER TABLE public.suppliers RENAME TO suppliers_legacy_0006;
    END IF;
END $$;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'suppliers_legacy_0006'
    )
    AND EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE connamespace = 'public'::regnamespace
          AND conname = 'suppliers_pkey'
          AND conrelid = 'public.suppliers_legacy_0006'::regclass
    ) THEN
        ALTER TABLE public.suppliers_legacy_0006 RENAME CONSTRAINT suppliers_pkey TO suppliers_legacy_0006_pkey;
    END IF;
END $$;

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

DO $$
BEGIN
    IF EXISTS (
        SELECT 1
        FROM information_schema.tables
        WHERE table_schema = 'public' AND table_name = 'suppliers_legacy_0006'
    ) THEN
        INSERT INTO public.suppliers (
            uuid,
            supplier_code,
            supplier_name,
            contact_person,
            contact_number,
            email_address,
            material_category,
            full_address,
            city,
            province,
            country,
            tax_id_npwp,
            bank_name,
            bank_account_number,
            bank_account_name,
            payment_terms,
            delivery_lead_time_days,
            status,
            created_at,
            updated_at
        )
        SELECT
            legacy.id,
            legacy.supplier_code,
            legacy.supplier_name,
            COALESCE(NULLIF(legacy.supplier_name, ''), 'Legacy Supplier'),
            '-',
            CONCAT('legacy+', legacy.id::text, '@local.invalid'),
            NULL,
            'Migrated from legacy schema',
            'Unknown',
            'Unknown',
            'Unknown',
            COALESCE(NULLIF(legacy.tax_id_npwp, ''), '-'),
            'Unknown',
            COALESCE(NULLIF(legacy.bank_account_number, ''), '-'),
            COALESCE(NULLIF(legacy.supplier_name, ''), 'Legacy Supplier'),
            COALESCE(NULLIF(legacy.payment_terms, ''), '-'),
            COALESCE(legacy.delivery_lead_time_days, 0),
            CASE
                WHEN legacy.status IN ('Active', 'Inactive') THEN legacy.status
                ELSE 'Active'
            END,
            COALESCE(legacy.created_at, NOW()),
            COALESCE(legacy.updated_at, NOW())
        FROM public.suppliers_legacy_0006 legacy
        WHERE NOT EXISTS (
            SELECT 1
            FROM public.suppliers current
            WHERE current.uuid = legacy.id
               OR current.supplier_code = legacy.supplier_code
        );
    END IF;
END $$;
