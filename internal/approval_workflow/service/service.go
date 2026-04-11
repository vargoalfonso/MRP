package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/approval_workflow/models"
	approvalWorkflowRepo "github.com/ganasa18/go-template/internal/approval_workflow/repository"
)

type IApprovalWorkflowService interface {
	GetAll(ctx context.Context) ([]models.ApprovalWorkflow, error)
	Create(ctx context.Context, req models.CreateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error)
	GetByID(ctx context.Context, id int64) (*models.ApprovalWorkflow, error)
	Update(ctx context.Context, id int64, req models.UpdateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo approvalWorkflowRepo.IApprovalWorkflowRepository
}

func New(repo approvalWorkflowRepo.IApprovalWorkflowRepository) IApprovalWorkflowService {
	return &service{repo: repo}
}

// =========================
// CRUD
// =========================
func (s *service) GetAll(ctx context.Context) ([]models.ApprovalWorkflow, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.ApprovalWorkflow, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error) {
	return s.repo.Create(ctx, req)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}
