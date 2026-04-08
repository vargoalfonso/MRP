-- Dijalankan otomatis oleh PostgreSQL saat volume pertama kali dibuat.
-- File ini di-mount ke /docker-entrypoint-initdb.d/
-- Migration files ada di sub-folder /docker-entrypoint-initdb.d/migrations/

\i /docker-entrypoint-initdb.d/migrations/0001_create_users_up.sql
