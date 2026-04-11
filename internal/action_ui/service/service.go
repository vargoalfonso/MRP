package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/action_ui/models"
	"github.com/ganasa18/go-template/internal/action_ui/repository"
)

type IService interface {
	CreateIncomingScan(ctx context.Context, req models.IncomingScanRequest, scannedBy string) (*models.IncomingScanResponse, bool, error)
}

type service struct {
	repo repository.IRepository
}

func New(repo repository.IRepository) IService {
	return &service{repo: repo}
}

func (s *service) CreateIncomingScan(ctx context.Context, req models.IncomingScanRequest, scannedBy string) (*models.IncomingScanResponse, bool, error) {
	return s.repo.CreateIncomingScan(ctx, req, scannedBy)
}
