-- =============================================================================
-- Migration: 0003_create_bom_tables_down.sql
-- Rollback BOM module tables (reverse dependency order)
-- =============================================================================

DROP TABLE IF EXISTS bom_closure                CASCADE;
DROP TABLE IF EXISTS bom_approvals              CASCADE;
DROP TABLE IF EXISTS bom_lines                  CASCADE;
DROP TABLE IF EXISTS bom_item                   CASCADE;
DROP TABLE IF EXISTS routing_operation_toolings CASCADE;
DROP TABLE IF EXISTS routing_operations         CASCADE;
DROP TABLE IF EXISTS routing_headers            CASCADE;
DROP TABLE IF EXISTS item_assets                CASCADE;
DROP TABLE IF EXISTS item_material_specs        CASCADE;
DROP TABLE IF EXISTS item_revisions             CASCADE;
DROP TABLE IF EXISTS items                      CASCADE;
DROP TABLE IF EXISTS suppliers                  CASCADE;
DROP TABLE IF EXISTS master_machines            CASCADE;
DROP TABLE IF EXISTS process_parameters         CASCADE;
DROP TABLE IF EXISTS uom_parameters             CASCADE;
