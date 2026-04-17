package service

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/skip2/go-qrcode"

	"github.com/ganasa18/go-template/internal/master_machine/models"
	machineRepo "github.com/ganasa18/go-template/internal/master_machine/repository"
)

type IMasterMachineService interface {
	GetAll(ctx context.Context) ([]models.MasterMachine, error)
	Create(ctx context.Context, req models.CreateMachineRequest) (*models.MasterMachine, error)
	GetByID(ctx context.Context, id int64) (*models.MasterMachine, error)
	Update(ctx context.Context, id int64, req models.UpdateMachineRequest) (*models.MasterMachine, error)
	Delete(ctx context.Context, id int64) error
	EnsureQR(ctx context.Context, id int64) (string, error)
}

type service struct {
	repo machineRepo.IMasterMachineRepository
}

func New(repo machineRepo.IMasterMachineRepository) IMasterMachineService {
	return &service{repo: repo}
}

func (s *service) GetAll(ctx context.Context) ([]models.MasterMachine, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) Create(ctx context.Context, req models.CreateMachineRequest) (*models.MasterMachine, error) {
	m, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	qr, err := generateMachineQRDataURL(m.ID, m.MachineNumber)
	if err != nil {
		return nil, err
	}

	if err := s.repo.UpdateQR(ctx, m.ID, qr); err != nil {
		return nil, err
	}

	m.QRImageBase64 = &qr
	return m, nil
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.MasterMachine, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateMachineRequest) (*models.MasterMachine, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) EnsureQR(ctx context.Context, id int64) (string, error) {
	m, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return "", err
	}

	if m.QRImageBase64 != nil && *m.QRImageBase64 != "" {
		return *m.QRImageBase64, nil
	}

	qr, err := generateMachineQRDataURL(m.ID, m.MachineNumber)
	if err != nil {
		return "", err
	}
	if err := s.repo.UpdateQR(ctx, m.ID, qr); err != nil {
		return "", err
	}
	return qr, nil
}

func generateMachineQRDataURL(id int64, _ string) (string, error) {
	// Keep payload minimal: machine identity only.
	payload := fmt.Sprintf(`{"t":"machine","id":%d}`, id)
	png, err := qrcode.Encode(payload, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(png), nil
}
