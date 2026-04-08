package service

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ganasa18/go-template/internal/employee/models"
	"github.com/ganasa18/go-template/pkg/logger"
)

func (s *service) Register(ctx context.Context, req models.EmployeeReq) (int, error) {
	if err := s.repo.InsertEmployee(ctx, req); err != nil {
		logger.FromContext(ctx).Error("InsertEmployee failed", slog.Any("error", err))
		return http.StatusInternalServerError, err
	}
	return http.StatusCreated, nil
}
