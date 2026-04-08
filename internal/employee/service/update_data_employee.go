package service

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/ganasa18/go-template/internal/employee/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/logger"
)

func (s *service) UpdateDataEmployee(ctx context.Context, req models.EmployeeReq) (int, error) {
	if err := s.repo.UpdateDataEmployee(ctx, req); err != nil {
		logger.FromContext(ctx).Error("UpdateDataEmployee failed",
			slog.String("employee_id", req.EmployeeID),
			slog.Any("error", err),
		)
		if _, ok := apperror.As(err); ok {
			return http.StatusNotFound, err
		}
		return http.StatusInternalServerError, err
	}
	return http.StatusOK, nil
}
