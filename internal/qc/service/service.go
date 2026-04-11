package service

import (
	"context"
	"math"

	qcRepo "github.com/ganasa18/go-template/internal/qc/repository"
)

type IService interface {
	ListTasks(ctx context.Context, f qcRepo.ListFilter) ([]qcRepo.TaskListRow, int64, error)
	StartTask(ctx context.Context, taskID int64, performedBy string) error
	ApproveIncoming(ctx context.Context, taskID int64, approvedQty, ngQty, scrapQty int, notes *string, defects []interface{}, scrapDisposition *string, performedBy string) error
	RejectIncoming(ctx context.Context, taskID int64, rejectedQty int, reason string, defects []interface{}, disposition *string, performedBy string) error
}

type service struct{ repo qcRepo.IRepository }

func New(repo qcRepo.IRepository) IService { return &service{repo: repo} }

func (s *service) ListTasks(ctx context.Context, f qcRepo.ListFilter) ([]qcRepo.TaskListRow, int64, error) {
	items, total, err := s.repo.ListTasks(ctx, f)
	return items, total, err
}

func (s *service) StartTask(ctx context.Context, taskID int64, performedBy string) error {
	_, err := s.repo.StartTask(ctx, taskID, performedBy)
	return err
}

func (s *service) ApproveIncoming(ctx context.Context, taskID int64, approvedQty, ngQty, scrapQty int, notes *string, defects []interface{}, scrapDisposition *string, performedBy string) error {
	return s.repo.ApproveIncoming(ctx, taskID, approvedQty, ngQty, scrapQty, notes, defects, scrapDisposition, performedBy)
}

func (s *service) RejectIncoming(ctx context.Context, taskID int64, rejectedQty int, reason string, defects []interface{}, disposition *string, performedBy string) error {
	return s.repo.RejectIncoming(ctx, taskID, rejectedQty, reason, defects, disposition, performedBy)
}

func TotalPages(total int64, limit int) int64 {
	if limit <= 0 {
		return 1
	}
	return int64(math.Ceil(float64(total) / float64(limit)))
}
