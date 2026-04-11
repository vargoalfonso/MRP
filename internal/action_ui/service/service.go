package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/action_ui/models"
	"github.com/ganasa18/go-template/internal/action_ui/repository"
)

type IService interface {
	// LookupByPackingNumber resolves QR scan result → DN context for UI auto-fill.
	LookupByPackingNumber(ctx context.Context, packingNumber, itemUniqCode string) (*models.IncomingScanDNItem, error)
	CreateIncomingScan(ctx context.Context, req models.IncomingScanRequest, scannedBy string) (*models.IncomingScanResponse, bool, error)
}

type service struct {
	repo repository.IRepository
}

func New(repo repository.IRepository) IService {
	return &service{repo: repo}
}

func (s *service) LookupByPackingNumber(ctx context.Context, packingNumber, itemUniqCode string) (*models.IncomingScanDNItem, error) {
	return s.repo.LookupByPackingNumber(ctx, packingNumber, itemUniqCode)
}

func (s *service) CreateIncomingScan(ctx context.Context, req models.IncomingScanRequest, scannedBy string) (*models.IncomingScanResponse, bool, error) {
	return s.repo.CreateIncomingScan(ctx, req, scannedBy)
}
