# Master Data Postman Documentation

## Files

- Import collection: [master-data.postman_collection.json](master-data.postman_collection.json)
- Import environment: [master-data-local.postman_environment.json](master-data-local.postman_environment.json)

## Supplier Type Reference

### Supplier master: `material_category`
Used by `POST /api/v1/suppliers` and `PUT /api/v1/suppliers/:id`.

Allowed values:
- `Raw Material`
- `Indirect Raw Material`
- `Subcon`

Example create payload:

```json
{
  "supplier_name": "PT Master Supplier",
  "contact_person": "Rina",
  "contact_number": "081234567890",
  "email_address": "supplier@example.com",
  "material_category": "Raw Material",
  "full_address": "Jl. Industri No. 1, Bekasi",
  "city": "Bekasi",
  "province": "Jawa Barat",
  "country": "Indonesia",
  "tax_id_npwp": "01.234.567.8-901.000",
  "bank_name": "BCA",
  "bank_account_number": "1234567890",
  "bank_account_name": "PT Master Supplier",
  "payment_terms": "30 Days",
  "delivery_lead_time_days": 7,
  "status": "Active"
}
```

### Supplier item master: `type`
Used by `POST /api/v1/supplier-items` and `PUT /api/v1/supplier-items/:id`.

Allowed values:
- `raw_material`
- `indirect`
- `subcon`

Important note:
- For `supplier-item` create/update, write payload fields are sent as strings.
- Numeric fields such as `quantity`, `weight`, and `pcs_per_kanban` are accepted as string values and parsed by the backend.

Example create payload:

```json
{
  "supplier_uuid": "{{supplier_uuid}}",
  "sebango_code": "ASM-LV7-001",
  "uniq_code": "SUPITEM-001",
  "type": "raw_material",
  "description": "Steel Plate",
  "quantity": "40",
  "uom": "pcs",
  "weight": "2.5",
  "pcs_per_kanban": "40",
  "customer_cycle": "Daily",
  "status": "active"
}
```

## Warehouse Documentation Status

Current backend status:
- There is no implemented `plant` module yet in this repository.
- `warehouse` backend is implemented and exposed at `/api/v1/warehouses`.
- `plant` backend is still not implemented, so `plant_id` is currently stored as a required string value.

## Warehouse API Contract

### Create warehouse
- `POST /api/v1/warehouses`

Payload:

```json
{
  "warehouse_name": "Main Raw Material Warehouse",
  "type_warehouse": "raw_material",
  "plant_id": "plant-uuid-or-code"
}
```

### Update warehouse
- `PUT /api/v1/warehouses/:id`

Payload:

```json
{
  "warehouse_name": "Main Raw Material Warehouse Updated",
  "type_warehouse": "finished_goods",
  "plant_id": "plant-uuid-or-code"
}
```

### Delete warehouse
- `DELETE /api/v1/warehouses/:id`

### List warehouse
- `GET /api/v1/warehouses`

Supported query params:
- `search`
- `type_warehouse`
- `plant_id`
- `page`
- `limit`

### Warehouse type values
- `raw_material`
- `wip`
- `finished_goods`
- `subcon`
- `general`

## Import Steps

1. Import [master-data.postman_collection.json](master-data.postman_collection.json)
2. Import [master-data-local.postman_environment.json](master-data-local.postman_environment.json)
3. Fill `auth_email` and `auth_password`
4. Run `Auth > Login`
5. Run `Supplier Master > Create Supplier`
6. Run `Supplier Item Master > Create Supplier Item`
7. Run `Warehouse Master > Create Warehouse`

