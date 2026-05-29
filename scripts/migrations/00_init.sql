-- Dijalankan otomatis oleh PostgreSQL saat volume pertama kali dibuat.
-- File ini di-mount ke /docker-entrypoint-initdb.d/
-- Migration files ada di sub-folder /docker-entrypoint-initdb.d/migrations/

\i /docker-entrypoint-initdb.d/migrations/0001_create_users_up.sql
\i /docker-entrypoint-initdb.d/migrations/0002_create_suppliers_up.sql
\i /docker-entrypoint-initdb.d/migrations/0003_create_customers_up.sql
\i /docker-entrypoint-initdb.d/migrations/0004_create_uniq_bill_of_materials_up.sql
\i /docker-entrypoint-initdb.d/migrations/0005_create_prls_up.sql
\i /docker-entrypoint-initdb.d/migrations/0006_sync_customer_and_supplier_schema_up.sql
\i /docker-entrypoint-initdb.d/migrations/0007_create_type_parameters_up.sql
\i /docker-entrypoint-initdb.d/migrations/0067_add_description_to_type_parameters_up.sql
