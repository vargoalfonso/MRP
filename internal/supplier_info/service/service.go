package service

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	siModels "github.com/ganasa18/go-template/internal/supplier_info/models"
	siRepo "github.com/ganasa18/go-template/internal/supplier_info/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/bulkimport"
	"github.com/google/uuid"
)

type SupplierInfoService interface {
	Create(ctx context.Context, req siModels.CreateSupplierInfoRequest) (*siModels.SupplierInfo, error)
	GetByUUID(ctx context.Context, id string) (*siModels.SupplierInfo, error)
	List(ctx context.Context, query siModels.ListSupplierInfoQuery) ([]siModels.SupplierInfo, int64, error)
	Update(ctx context.Context, id string, req siModels.UpdateSupplierInfoRequest) (*siModels.SupplierInfo, error)
	Delete(ctx context.Context, id string) error

	// Bulk import via Excel.
	ImportFromExcel(ctx context.Context, r io.Reader, filename string) (bulkimport.BulkResult, error)
	DownloadImportTemplate(ctx context.Context) ([]byte, error)
	DownloadImportErrors(ctx context.Context, token string) ([]byte, error)
}

type svc struct {
	repo siRepo.IRepository

	errorStoreMu sync.Mutex
	errorStore   bulkimport.ErrorStore
}

func New(repo siRepo.IRepository) SupplierInfoService {
	return &svc{repo: repo}
}

// normalizeType maps supplier_item.type values to display labels
func normalizeTypeLabel(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "raw_material":
		return "RM"
	case "indirect":
		return "IRM"
	case "subcon":
		return "SUBCON"
	default:
		return strings.ToUpper(strings.TrimSpace(t))
	}
}

// itemTypeFromLabel memetakan label tampilan (RM/IRM/SUBCON) kembali ke
// nilai supplier_item.type. Nilai yang sudah berupa tipe asli ikut diterima.
func itemTypeFromLabel(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "rm", "raw_material":
		return "raw_material"
	case "irm", "indirect":
		return "indirect"
	case "subcon":
		return "subcon"
	default:
		return strings.ToLower(strings.TrimSpace(t))
	}
}

func toOptStr(s string) *string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return &s
}

func (s *svc) Create(ctx context.Context, req siModels.CreateSupplierInfoRequest) (*siModels.SupplierInfo, error) {
	var (
		supplierName string
		typeLabel    string
		key          string
		keyLabel     string
	)

	if mc := strings.TrimSpace(req.MaterialCode); mc != "" {
		// Mode Material Code: kunci mapping = material code (disimpan di kolom
		// uniq). Cukup pastikan supplier + type punya supplier_item aktif.
		itemType := itemTypeFromLabel(req.Type)
		supplierName = strings.TrimSpace(req.SupplierName)
		typeLabel = normalizeTypeLabel(itemType)
		ok, err := s.repo.ExistsActiveSupplierItem(ctx, supplierName, itemType)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, apperror.NotFound("kombinasi supplier dan type tidak ditemukan di supplier item")
		}
		key, keyLabel = mc, "UNIQ"
	} else {
		// Mode lama: cari supplier_item sesuai pilihan user (uniq + supplier + type).
		// Jangan hanya berdasarkan uniq: UNIQ yang sama bisa dimiliki beberapa supplier/type.
		item, err := s.repo.FindSupplierItem(ctx, req.Uniq, req.SupplierName, itemTypeFromLabel(req.Type))
		if err != nil {
			return nil, err
		}
		supplierName = item.SupplierName
		typeLabel = normalizeTypeLabel(item.Type)
		key, keyLabel = strings.TrimSpace(req.Uniq), "UNIQ"
	}

	// Duplikat dicek per kombinasi (kunci, supplier, type)
	exists, err := s.repo.ExistsByUniqSupplierType(ctx, key, supplierName, typeLabel)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.BadRequest(fmt.Sprintf("%s '%s' untuk %s (%s) sudah terdaftar di supplier info",
			keyLabel, strings.ToUpper(key), supplierName, typeLabel))
	}

	info := &siModels.SupplierInfo{
		UUID:         uuid.NewString(),
		Uniq:         key,
		UniqZahir:    toOptStr(req.UniqZahir),
		SupplierName: supplierName,
		Type:         typeLabel,
		Status:       strings.ToLower(strings.TrimSpace(req.Status)),
	}

	if err := s.repo.Create(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *svc) GetByUUID(ctx context.Context, id string) (*siModels.SupplierInfo, error) {
	if strings.TrimSpace(id) == "" {
		return nil, apperror.BadRequest("id is required")
	}
	return s.repo.FindByUUID(ctx, id)
}

func (s *svc) List(ctx context.Context, query siModels.ListSupplierInfoQuery) ([]siModels.SupplierInfo, int64, error) {
	return s.repo.List(ctx, query)
}

func (s *svc) Update(ctx context.Context, id string, req siModels.UpdateSupplierInfoRequest) (*siModels.SupplierInfo, error) {
	info, err := s.repo.FindByUUID(ctx, id)
	if err != nil {
		return nil, err
	}
	info.UniqZahir = toOptStr(req.UniqZahir)
	info.Status = strings.ToLower(strings.TrimSpace(req.Status))
	if err := s.repo.Update(ctx, info); err != nil {
		return nil, err
	}
	return info, nil
}

func (s *svc) Delete(ctx context.Context, id string) error {
	info, err := s.repo.FindByUUID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, info)
}
