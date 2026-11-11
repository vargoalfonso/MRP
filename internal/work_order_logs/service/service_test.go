package service

import (
	"context"
	"testing"

	"github.com/ganasa18/go-template/internal/work_order_logs/models"
)

type fakeRepo struct {
	created []*models.WorkOrderAutomationLog
	rows    []models.WorkOrderAutomationLog
	gotF    models.ListLogFilters
}

func (f *fakeRepo) Create(_ context.Context, log *models.WorkOrderAutomationLog) error {
	f.created = append(f.created, log)
	return nil
}

func (f *fakeRepo) List(_ context.Context, flt models.ListLogFilters) ([]models.WorkOrderAutomationLog, int64, error) {
	f.gotF = flt
	return f.rows, int64(len(f.rows)), nil
}

func TestCreate_DefaultsAndTrim(t *testing.T) {
	repo := &fakeRepo{}
	s := New(repo)

	resp, err := s.Create(context.Background(), models.CreateLogRequest{
		WoNumber:      "  WO-231026-001 ",
		FailureReason: "uniq tidak ditemukan",
	})
	if err != nil {
		t.Fatalf("Create error: %v", err)
	}
	if len(repo.created) != 1 {
		t.Fatalf("created = %d; want 1", len(repo.created))
	}
	got := repo.created[0]
	if got.WoNumber != "WO-231026-001" {
		t.Errorf("wo_number = %q; want trimmed", got.WoNumber)
	}
	if got.Source != "automation" || got.Status != "failed" {
		t.Errorf("source/status = %q/%q; want automation/failed", got.Source, got.Status)
	}
	if got.UUID == "" || resp.ID != got.UUID {
		t.Errorf("uuid tidak konsisten: entity=%q resp=%q", got.UUID, resp.ID)
	}
}

func TestList_PaginationAndFilters(t *testing.T) {
	repo := &fakeRepo{rows: []models.WorkOrderAutomationLog{{UUID: "a"}, {UUID: "b"}}}
	s := New(repo)

	resp, err := s.List(context.Background(), models.ListLogQuery{WoNumber: "WO-1", Page: 3, Limit: 9999})
	if err != nil {
		t.Fatalf("List error: %v", err)
	}
	if repo.gotF.Limit != 200 {
		t.Errorf("limit = %d; want clamp 200", repo.gotF.Limit)
	}
	if repo.gotF.Offset != 400 {
		t.Errorf("offset = %d; want 400", repo.gotF.Offset)
	}
	if repo.gotF.WoNumber != "WO-1" {
		t.Errorf("wo_number filter = %q", repo.gotF.WoNumber)
	}
	if len(resp.Items) != 2 || resp.Pagination.Total != 2 {
		t.Errorf("items/total = %d/%d; want 2/2", len(resp.Items), resp.Pagination.Total)
	}

	if _, err := s.List(context.Background(), models.ListLogQuery{}); err != nil {
		t.Fatal(err)
	}
	if repo.gotF.Limit != 20 || repo.gotF.Offset != 0 {
		t.Errorf("default limit/offset = %d/%d; want 20/0", repo.gotF.Limit, repo.gotF.Offset)
	}
}
