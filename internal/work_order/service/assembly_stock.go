package service

import (
	"strings"

	woModels "github.com/ganasa18/go-template/internal/work_order/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/assemblystock"
	"gorm.io/gorm"
)

const workOrderTypeAssembly = "Assembly"

func isAssemblyWO(woType string) bool {
	return strings.EqualFold(strings.TrimSpace(woType), workOrderTypeAssembly)
}

func assemblyRequestsFromItems(items []woModels.CreateWorkOrderItem) []assemblystock.Request {
	out := make([]assemblystock.Request, 0, len(items))
	for _, it := range items {
		out = append(out, assemblystock.Request{ParentUniq: it.ItemUniqCode, Qty: it.Quantity})
	}
	return out
}

func assemblyRequestsFromBulkItems(items []woModels.CreateBulkWorkOrderItem) []assemblystock.Request {
	out := make([]assemblystock.Request, 0, len(items))
	for _, it := range items {
		out = append(out, assemblystock.Request{ParentUniq: it.ItemUniqCode, Qty: it.Quantity})
	}
	return out
}

// planAssemblyStock menjalankan Plan dan menerjemahkan hasilnya ke AppError.
func planAssemblyStock(db *gorm.DB, reqs []assemblystock.Request, lock bool) ([]assemblystock.Allocation, error) {
	allocs, shortages, err := assemblystock.Plan(db, reqs, lock)
	if err != nil {
		if assemblystock.IsNoBOM(err) {
			return nil, apperror.UnprocessableEntity(err.Error())
		}
		return nil, apperror.InternalWrap("assembly stock check", err)
	}
	if len(shortages) > 0 {
		return nil, apperror.UnprocessableEntity(assemblystock.FormatShortages(shortages))
	}
	return allocs, nil
}

// checkAssemblyStock hanya memeriksa (tanpa lock). Dipakai Preview.
func (s *service) checkAssemblyStock(woType string, reqs []assemblystock.Request) error {
	if !isAssemblyWO(woType) {
		return nil
	}
	_, err := planAssemblyStock(s.db, reqs, false)
	return err
}

// reserveAssemblyStock memeriksa dan mengunci stok komponen untuk WO Assembly di
// dalam transaksi pembuatan WO. Gagal -> transaksi di-rollback, WO tidak terbentuk.
func (s *service) reserveAssemblyStock(tx *gorm.DB, woID int64, woNumber, woType string, reqs []assemblystock.Request) error {
	if !isAssemblyWO(woType) {
		return nil
	}
	allocs, err := planAssemblyStock(tx, reqs, true)
	if err != nil {
		return err
	}
	if err := assemblystock.Lock(tx, woID, woNumber, allocs); err != nil {
		return apperror.InternalWrap("assembly stock lock", err)
	}
	return nil
}
