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
