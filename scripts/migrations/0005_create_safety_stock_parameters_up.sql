CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE
    IF NOT EXISTS safety_stock_parameters (
        id BIGSERIAL PRIMARY KEY,
        inventory_type VARCHAR(50) NOT NULL,
        item_uniq_code VARCHAR(100) NOT NULL,
        calculation_type VARCHAR(50) NOT NULL,
        constanta DOUBLE PRECISION DEFAULT 0,
        created_at TIMESTAMPTZ DEFAULT NOW (),
        updated_at TIMESTAMPTZ DEFAULT NOW ()
    );
