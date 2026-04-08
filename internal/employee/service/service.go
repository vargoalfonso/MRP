package service

import (
	"context"

	"github.com/ganasa18/go-template/internal/employee/models"
	employeeRepository "github.com/ganasa18/go-template/internal/employee/repository"
	"github.com/ganasa18/go-template/pkg/httpclient"
)

// Service defines the employee business-logic contract.
type Service interface {
	GetAllDataEmployee(ctx context.Context) ([]models.EmployeeResp, int, error)
	GetDataEmployeeByID(ctx context.Context, id string) (models.EmployeeResp, int, error)
	Register(ctx context.Context, req models.EmployeeReq) (int, error)
	UpdateDataEmployee(ctx context.Context, req models.EmployeeReq) (int, error)
}

type service struct {
	repo       employeeRepository.IRepository
	httpClient httpclient.Client
}

// NewService constructs an employee Service.
func NewService(repo employeeRepository.IRepository, httpClient httpclient.Client) Service {
	return &service{repo: repo, httpClient: httpClient}
}
