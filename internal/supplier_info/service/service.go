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
	// Cari supplier_item sesuai pilihan user (uniq + supplier + type). Jangan
	// hanya berdasarkan uniq: UNIQ yang sama bisa dimiliki beberapa supplier/type.
	item, err := s.repo.FindSupplierItem(ctx, req.Uniq, req.SupplierName, itemTypeFromLabel(req.Type))
	if err != nil {
		return nil, err
	}

	supplierName := item.SupplierName
	typeLabel := normalizeTypeLabel(item.Type)

	// Duplikat dicek per kombinasi (uniq, supplier, type)
	exists, err := s.repo.ExistsByUniqSupplierType(ctx, req.Uniq, supplierName, typeLabel)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, apperror.BadRequest(fmt.Sprintf("UNIQ '%s' untuk %s (%s) sudah terdaftar di supplier info",
			strings.ToUpper(strings.TrimSpace(req.Uniq)), supplierName, typeLabel))
	}

	info := &siModels.SupplierInfo{
		UUID:         uuid.NewString(),
		Uniq:         strings.TrimSpace(req.Uniq),
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
