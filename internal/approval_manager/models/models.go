package models

import awmodels "github.com/ganasa18/go-template/internal/approval_workflow/models"

type SummaryResponse = awmodels.ApprovalManagerSummary
type Pagination = awmodels.ApprovalManagerPagination
type Item = awmodels.ApprovalManagerItem
type ListResponse = awmodels.ApprovalManagerListResponse
type DetailResponse = awmodels.ApprovalManagerDetail

type ListQuery struct {
	Type         string
	Status       string
	Search       string
	SubmittedBy  string
	CurrentLevel int
	Scope        string
	Page         int
	Limit        int
	Offset       int
}
