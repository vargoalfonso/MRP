ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_employee;

DROP TABLE IF EXISTS access_control_matrix;
DROP TABLE IF EXISTS employees;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS departments;
