package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/ganasa18/go-template/internal/supplier_item/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

// typeMaterialsFor memetakan type supplier item ke nilai item_material_specs.type_material.
func typeMaterialsFor(itemType string) ([]string, error) {
	switch normalizeType(itemType) {
	case "raw_material":
		return []string{"raw", "raw_material", ""}, nil
	case "indirect":
		return []string{"indirect", "indirect_raw_material"}, nil
	default:
		return nil, apperror.BadRequest("type harus raw_material atau indirect untuk pendaftaran per material code")
	}
}

// PreviewByMaterialCode: daftar uniq yang akan ikut kalau material code ini
// dipilih untuk supplier tertentu (sekaligus menandai yang sudah terdaftar).
func (s *service) PreviewByMaterialCode(ctx context.Context, q models.MaterialCodePreviewQuery) ([]models.MaterialCodeUniq, error) {
	code := strings.TrimSpace(q.MaterialCode)
	if code == "" {
		return nil, apperror.BadRequest("material_code is required")
	}
	typeMaterials, err := typeMaterialsFor(q.Type)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListUniqsByMaterialCode(ctx, code, typeMaterials)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(q.SupplierUUID) == "" || len(rows) == 0 {
		return rows, nil
	}

	// Supplier sudah punya baris lama (uniq_code = nama master) untuk material
	// code ini: itu sudah berlaku untuk semua uniq-nya -> jangan buat dobel.
	legacy, err := s.repo.SupplierHasLegacyMaterialRow(ctx, q.SupplierUUID, q.Type, code)
	if err != nil {
		return nil, err
	}
	if legacy {
		for i := range rows {
			rows[i].AlreadyListed = true
		}
		return rows, nil
	}

	codes := make([]string, 0, len(rows))
	for _, r := range rows {
		codes = append(codes, r.UniqCode)
	}
	registered, err := s.repo.ListRegisteredUniqs(ctx, q.SupplierUUID, q.Type, codes)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		rows[i].AlreadyListed = registered[strings.ToUpper(strings.TrimSpace(rows[i].UniqCode))]
	}
	return rows, nil
}

// CreateByMaterialCode mendaftarkan SEMUA uniq ber-material-code tertentu ke
// satu supplier. Uniq yang sudah terdaftar untuk supplier+type tsb dilewati.
// Semua baris baru disimpan dalam satu transaksi.
func (s *service) CreateByMaterialCode(ctx context.Context, req models.BulkByMaterialCodeRequest) (*models.BulkByMaterialCodeResult, error) {
	supplier, err := s.repo.FindSupplierByUUID(ctx, req.SupplierUUID)
	if err != nil {
		return nil, err
	}
	itemType := normalizeType(req.Type)
	typeMaterials, err := typeMaterialsFor(itemType)
	if err != nil {
		return nil, err
	}
	code := strings.TrimSpace(req.MaterialCode)

	legacy, err := s.repo.SupplierHasLegacyMaterialRow(ctx, supplier.UUID, itemType, code)
	if err != nil {
		return nil, err
	}
	if legacy {
		return nil, apperror.BadRequest("supplier ini sudah terdaftar untuk material code '" + code +
			"' (data lama berbasis nama master) dan sudah berlaku untuk semua uniq dengan material code tersebut")
	}

	quantity, err := parseRequiredInt64(req.Quantity, "quantity")
	if err != nil {
		return nil, err
	}
	warehouseUUID, warehouseName, err := s.resolveWarehouse(ctx, req.WarehouseUUID)
	if err != nil {
		return nil, err
	}

	all, err := s.repo.ListUniqsByMaterialCode(ctx, code, typeMaterials)
	if err != nil {
		return nil, err
	}

	// Batasi ke subset pilihan user kalau ada.
	if len(req.SelectedUniqCodes) > 0 {
		want := make(map[string]bool, len(req.SelectedUniqCodes))
		for _, u := range req.SelectedUniqCodes {
			want[strings.ToUpper(strings.TrimSpace(u))] = true
		}
		filtered := make([]models.MaterialCodeUniq, 0, len(all))
		for _, r := range all {
			if want[strings.ToUpper(strings.TrimSpace(r.UniqCode))] {
				filtered = append(filtered, r)
			}
		}
		all = filtered
	}
	if len(all) == 0 {
		return nil, apperror.NotFound("tidak ada uniq dengan material code '" + code + "'")
	}

	codes := make([]string, 0, len(all))
	for _, r := range all {
		codes = append(codes, r.UniqCode)
	}
	registered, err := s.repo.ListRegisteredUniqs(ctx, supplier.UUID, itemType, codes)
	if err != nil {
		return nil, err
	}

	result := &models.BulkByMaterialCodeResult{
		MaterialCode:     code,
		Matched:          len(all),
		CreatedUniqCodes: []string{},
		SkippedUniqCodes: []string{},
	}
	toCreate := make([]*models.SupplierItem, 0, len(all))

	for _, r := range all {
		key := strings.ToUpper(strings.TrimSpace(r.UniqCode))
		if registered[key] {
			result.SkippedUniqCodes = append(result.SkippedUniqCodes, r.UniqCode)
			continue
		}

		uom := strings.TrimSpace(req.UOM)
		if uom == "" {
			uom = r.UOM
		}
		desc := r.PartName

		item := &models.SupplierItem{
			UUID:          uuid.NewString(),
			SupplierUUID:  supplier.UUID,
			SupplierName:  supplier.SupplierName,
			WarehouseUUID: warehouseUUID,
			WarehouseName: warehouseName,
			SebangoCode:   models.NormalizeOptionalString(toOptionalString(code)),
			UniqCode:      models.NormalizeOptionalString(toOptionalString(r.UniqCode)),
			Type:          itemType,
			Description:   models.NormalizeOptionalString(toOptionalString(desc)),
			Quantity:      quantity,
			UOM:           models.NormalizeOptionalString(toOptionalString(uom)),
			Weight:        r.WeightKg,
			PayloadJSON:   buildBulkPayload(itemType, r),
			Status:        normalizeStatus(req.Status),
		}
		toCreate = append(toCreate, item)
		result.CreatedUniqCodes = append(result.CreatedUniqCodes, r.UniqCode)
	}

	if err := s.repo.CreateMany(ctx, toCreate); err != nil {
		return nil, err
	}
	result.Created = len(toCreate)
	result.Skipped = len(result.SkippedUniqCodes)
	return result, nil
}

// buildBulkPayload membuat payload_json yang bentuknya sama dengan yang
// dikirim form create, supaya halaman list/edit membacanya seperti biasa.
func buildBulkPayload(itemType string, r models.MaterialCodeUniq) datatypes.JSON {
	payload := map[string]interface{}{
		"type":        itemType,
		"uniq_code":   r.UniqCode,
		"part_name":   r.PartName,
		"part_number": r.PartNumber,
		"grade":       r.Grade,
		"payload_detail": map[string]interface{}{
			"material_spec": map[string]interface{}{
				"material_grade": r.MaterialCode,
				"grade":          r.Grade,
				"form":           r.Form,
				"width_mm":       r.WidthMm,
				"diameter_mm":    r.DiameterMm,
				"thickness_mm":   r.ThicknessMm,
				"length_mm":      r.LengthMm,
				"weight_kg":      r.WeightKg,
			},
		},
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return nil
	}
	return datatypes.JSON(b)
}

// normalizeMasterName: huruf besar + spasi dirapikan; harus sama dengan
// normalisasi di SQL (UPPER(REGEXP_REPLACE(TRIM(x), '\s+', ' ', 'g'))).
func normalizeMasterName(value string) string {
	return strings.ToUpper(strings.Join(strings.Fields(value), " "))
}

// resolveLegacyMaterialCodes memperbaiki data LAMA saat dibaca (tidak mengubah
// database): form create dulu menyimpan NAMA master (mis. "BR50 WIRE") ke
// uniq_code dan sebango_code. Kalau uniq_code sebuah item raw/indirect sama
// dengan nama master yang gradenya tunggal, sebango_code dikembalikan sebagai
// material code (grade, mis. "BR50") -- sehingga pencarian supplier berdasarkan
// material code (PO Budget dsb.) langsung cocok untuk SEMUA material.
// setter dipanggil dengan indeks item dan material code hasil terjemahan.
func (s *service) resolveLegacyMaterialCodes(ctx context.Context, items []models.SupplierItem, setter func(i int, code string)) {
	if len(items) == 0 {
		return
	}
	keys := make([]string, 0, len(items))
	seen := make(map[string]bool, len(items))
	for _, it := range items {
		if it.Type != "raw_material" && it.Type != "indirect" {
			continue
		}
		if it.UniqCode == nil {
			continue
		}
		k := normalizeMasterName(*it.UniqCode)
		if k == "" || seen[k] {
			continue
		}
		seen[k] = true
		keys = append(keys, k)
	}
	if len(keys) == 0 {
		return
	}
	grades, err := s.repo.ResolveMasterGradesByName(ctx, keys)
	if err != nil || len(grades) == 0 {
		return // best-effort: data tetap apa adanya bila lookup gagal
	}
	for i, it := range items {
		if (it.Type != "raw_material" && it.Type != "indirect") || it.UniqCode == nil {
			continue
		}
		uniqKey := normalizeMasterName(*it.UniqCode)
		grade, ok := grades[uniqKey]
		if !ok {
			continue
		}
		// Hanya timpa bila sebango_code kosong atau cuma salinan uniq_code (nama master).
		if it.SebangoCode == nil || normalizeMasterName(*it.SebangoCode) == "" ||
			normalizeMasterName(*it.SebangoCode) == uniqKey {
			setter(i, grade)
		}
	}
}
