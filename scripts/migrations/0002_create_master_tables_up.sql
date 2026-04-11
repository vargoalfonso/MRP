-- =========================
-- EXTENSION
-- =========================
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =========================
-- TABLE: departments
-- =========================
CREATE TABLE
 IF NOT EXISTS departments (
  id BIGSERIAL PRIMARY KEY,
  department_code VARCHAR(50),
  department_name VARCHAR(100),
  description TEXT,
  manager_id BIGINT,
  parent_department_id BIGINT,
  status VARCHAR(20),
  created_at TIMESTAMPTZ DEFAULT NOW (),
  updated_at TIMESTAMPTZ DEFAULT NOW ()
 );

-- =========================
-- TABLE: roles
-- =========================
CREATE TABLE
 IF NOT EXISTS roles (
  id BIGSERIAL PRIMARY KEY,
  name VARCHAR(100),
  description TEXT,
  permissions JSONB,
  status VARCHAR(20),
  created_at TIMESTAMPTZ DEFAULT NOW (),
  updated_at TIMESTAMPTZ DEFAULT NOW ()
 );

-- =========================
-- TABLE: employees (FIXED)
-- =========================
CREATE TABLE
 IF NOT EXISTS employees (
  id BIGSERIAL PRIMARY KEY,
  full_name VARCHAR(150),
  email VARCHAR(150),
  phone_number VARCHAR(50),
  job_title VARCHAR(100),
  unit_cost NUMERIC(15, 2),
  join_date DATE,
  role_id BIGINT,
  department_id BIGINT,
  reports_to_id BIGINT,
  status VARCHAR(20),
  notes TEXT,
  created_at TIMESTAMPTZ DEFAULT NOW (),
  updated_at TIMESTAMPTZ DEFAULT NOW (),
  CONSTRAINT fk_employee_role FOREIGN KEY (role_id) REFERENCES roles (id),
  CONSTRAINT fk_employee_department FOREIGN KEY (department_id) REFERENCES departments (id),
  CONSTRAINT fk_employee_manager FOREIGN KEY (reports_to_id) REFERENCES employees (id)
 );

-- =========================
-- TABLE: access_control_matrices
-- =========================
CREATE TABLE
 IF NOT EXISTS access_control_matrices (
  id BIGSERIAL PRIMARY KEY,
  full_name VARCHAR(150),
  employee_id BIGINT,
  role_id BIGINT,
  department_id BIGINT,
  status VARCHAR(20),
  created_at TIMESTAMPTZ DEFAULT NOW (),
  updated_at TIMESTAMPTZ DEFAULT NOW (),
  CONSTRAINT fk_acm_employee FOREIGN KEY (employee_id) REFERENCES employees (id),
  CONSTRAINT fk_acm_role FOREIGN KEY (role_id) REFERENCES roles (id),
  CONSTRAINT fk_acm_department FOREIGN KEY (department_id) REFERENCES departments (id)
 );
