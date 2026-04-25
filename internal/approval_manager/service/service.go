package service

import (
	"context"
	"strings"

	approvalModels "github.com/ganasa18/go-template/internal/approval_manager/models"
	"github.com/ganasa18/go-template/internal/approval_manager/repository"
)

type IService interface {
	GetSummary(ctx context.Context, filterType string) (*approvalModels.SummaryResponse, error)
	ListItems(ctx context.Context, q approvalModels.ListQuery, userRoles []string) (*approvalModels.ListResponse, error)
	GetDetail(ctx context.Context, instanceID int64, userRoles []string) (*approvalModels.DetailResponse, error)
}

type service struct{ repo repository.IRepository }

func New(repo repository.IRepository) IService { return &service{repo: repo} }

func (s *service) GetSummary(ctx context.Context, filterType string) (*approvalModels.SummaryResponse, error) {
	return s.repo.GetSummary(ctx, filterType)
}

func (s *service) ListItems(ctx context.Context, q approvalModels.ListQuery, userRoles []string) (*approvalModels.ListResponse, error) {
	items, total, err := s.repo.ListItems(ctx, q)
	if err != nil {
		return nil, err
	}
	filtered := make([]approvalModels.Item, 0, len(items))
	for _, item := range items {
		applyPermissions(&item, userRoles)
		if strings.EqualFold(strings.TrimSpace(q.Scope), "my_turn") && !item.IsMyTurn {
			continue
		}
		filtered = append(filtered, item)
	}
	return &approvalModels.ListResponse{Items: filtered, Pagination: s.repo.BuildPagination(total, q.Page, q.Limit)}, nil
}

func (s *service) GetDetail(ctx context.Context, instanceID int64, userRoles []string) (*approvalModels.DetailResponse, error) {
	resp, err := s.repo.GetDetail(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	applyDetailPermissions(resp, userRoles)
	return resp, nil
}

func applyPermissions(item *approvalModels.Item, userRoles []string) {
	item.CanView = true
	item.IsMyTurn = item.Status == "pending" && hasRole(userRoles, item.CurrentLevelRole)
	item.CanApprove = item.IsMyTurn
	item.CanReject = item.IsMyTurn
	item.ViewMode = "read_only"
	if item.IsMyTurn {
		item.ViewMode = "actionable"
	}
}

func applyDetailPermissions(item *approvalModels.DetailResponse, userRoles []string) {
	levelRole := ""
	switch item.CurrentLevel {
	case 1:
		levelRole = item.Workflow.Level1Role
	case 2:
		levelRole = item.Workflow.Level2Role
	case 3:
		levelRole = item.Workflow.Level3Role
	case 4:
		levelRole = item.Workflow.Level4Role
	}
	item.CanView = true
	item.IsMyTurn = item.Status == "pending" && hasRole(userRoles, levelRole)
	item.CanApprove = item.IsMyTurn
	item.CanReject = item.IsMyTurn
	item.ViewMode = "read_only"
	if item.IsMyTurn {
		item.ViewMode = "actionable"
	}
}

func hasRole(userRoles []string, required string) bool {
	required = strings.TrimSpace(required)
	if required == "" {
		return false
	}
	for _, role := range userRoles {
		if strings.TrimSpace(role) == required {
			return true
		}
	}
	return false
}
