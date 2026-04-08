package service

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ganasa18/go-template/internal/employee/models"
	"github.com/ganasa18/go-template/pkg/logger"
)

func (s *service) GetAllDataEmployee(ctx context.Context) ([]models.EmployeeResp, int, error) {
	data, err := s.repo.GetAllDataEmployee(ctx)
	if err != nil {
		logger.FromContext(ctx).Error("GetAllDataEmployee failed", slog.Any("error", err))
		return nil, http.StatusInternalServerError, err
	}
	logger.FromContext(ctx).Info("GetAllDataEmployee success", slog.Int("count", len(data)))
	return data, http.StatusOK, nil
}

func (s *service) GetDataEmployeeByID(ctx context.Context, id string) (models.EmployeeResp, int, error) {
	data, err := s.repo.GetDataEmployeeByID(ctx, id)
	if err != nil {
		logger.FromContext(ctx).Error("GetDataEmployeeByID failed",
			slog.String("employee_id", id),
			slog.Any("error", err),
		)
		return models.EmployeeResp{}, http.StatusNotFound, err
	}
	return data, http.StatusOK, nil
}
