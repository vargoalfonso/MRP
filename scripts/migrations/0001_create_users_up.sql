CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE
    IF NOT EXISTS public.users (
        id BIGINT GENERATED ALWAYS AS IDENTITY,
        uuid UUID NOT NULL DEFAULT uuid_generate_v4 (),
        username VARCHAR(100) NOT NULL,
        email VARCHAR(255) NOT NULL,
        password VARCHAR(255) NOT NULL DEFAULT '',
        roles VARCHAR(255) NOT NULL DEFAULT 'user',
        employee_id BIGINT NULL,
        -- OAuth / SSO support
        provider VARCHAR(50) NOT NULL DEFAULT 'local',
        provider_id VARCHAR(255) NULL,
        avatar_url TEXT NULL,
        is_verified BOOLEAN NOT NULL DEFAULT FALSE,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW (),
        deleted_at TIMESTAMPTZ NULL,
        CONSTRAINT users_pkey PRIMARY KEY (id),
        CONSTRAINT users_uuid_key UNIQUE (uuid),
        CONSTRAINT users_username_key UNIQUE (username),
        CONSTRAINT users_email_key UNIQUE (email)
    );

CREATE INDEX IF NOT EXISTS idx_users_provider_id ON public.users (provider_id)
WHERE
    provider_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON public.users (deleted_at);
