package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ganasa18/go-template/internal/customer_order/models"
)

type fakeRepository struct {
	findByUUIDResult *models.CustomerOrderDocument
	findByUUIDErr    error
	summaryResult    models.SummaryResponse
	summaryErr       error
	summaryCalled    bool
	summaryType      string
	customerName     string
	customerNameErr  error
	itemSnapshots    map[string]*models.ItemSnapshot
	itemSnapshotErr  error
	updatedDoc       *models.CustomerOrderDocument
	updateErr        error
}

func (f *fakeRepository) Create(context.Context, *models.CustomerOrderDocument) error { return nil }

func (f *fakeRepository) FindByUUID(context.Context, string) (*models.CustomerOrderDocument, error) {
	return f.findByUUIDResult, f.findByUUIDErr
}

func (f *fakeRepository) List(context.Context, models.ListFilters) ([]models.CustomerOrderDocument, int64, error) {
	return nil, 0, nil
}

func (f *fakeRepository) UpdateStatus(context.Context, *models.CustomerOrderDocument) error {
	return nil
}

func (f *fakeRepository) Update(_ context.Context, doc *models.CustomerOrderDocument) error {
	if f.updateErr != nil {
		return f.updateErr
	}
	f.updatedDoc = docCloneForTest(doc)
	return nil
}

func (f *fakeRepository) SoftDelete(context.Context, *models.CustomerOrderDocument) error { return nil }

func (f *fakeRepository) GetActivePeriode(context.Context) (string, error) { return "", nil }

func (f *fakeRepository) GetCustomerNameByID(context.Context, int64) (string, error) {
	if f.customerNameErr != nil {
		return "", f.customerNameErr
	}
	return f.customerName, nil
}

func (f *fakeRepository) GetItemSnapshot(_ context.Context, uniqCode string) (*models.ItemSnapshot, error) {
	if f.itemSnapshotErr != nil {
		return nil, f.itemSnapshotErr
	}
	return f.itemSnapshots[uniqCode], nil
}

func (f *fakeRepository) GetSummary(_ context.Context, documentType string) (*models.SummaryResponse, error) {
	f.summaryCalled = true
	f.summaryType = documentType
	if f.summaryErr != nil {
		return nil, f.summaryErr
	}
	resp := f.summaryResult
	return &resp, nil
}

func TestGetByUUIDReturnsComputedDetailMetrics(t *testing.T) {
	repo := &fakeRepository{
		findByUUIDResult: &models.CustomerOrderDocument{
			UUID:                 "doc-1",
			DocumentType:         "DN",
			DocumentNumber:       "DN-TMC-2026-0001",
			DocumentDate:         time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
			CustomerID:           1,
			CustomerNameSnapshot: "Toyota",
			Status:               "active",
			Items: []models.CustomerOrderDocumentItem{
				{UUID: "item-1", LineNo: 1, ItemUniqCode: "LV-001", PartName: "Steel Plate", PartNumber: "SP-001-A", Quantity: 120},
				{UUID: "item-2", LineNo: 2, ItemUniqCode: "LV-002", PartName: "Steel Plate X", PartNumber: "SP-001-B", Quantity: 80},
			},
		},
	}

	svc := New(repo)

	resp, err := svc.GetByUUID(context.Background(), "doc-1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if resp.TotalQuantity != 200 {
		t.Fatalf("expected total quantity 200, got %v", resp.TotalQuantity)
	}
	if resp.TotalUniq != 2 {
		t.Fatalf("expected total uniq 2, got %d", resp.TotalUniq)
	}
	if resp.HeaderDeliveryDate == nil {
		t.Fatal("expected header delivery date for DN")
	}
}

func TestGetSummaryReturnsRepositoryResponse(t *testing.T) {
	repo := &fakeRepository{
		summaryResult: models.SummaryResponse{
			DocumentType:   "PO",
			TotalDocuments: 4,
			TotalQuantity:  900,
		},
	}

	svc := New(repo)

	resp, err := svc.GetSummary(context.Background(), models.SummaryRequest{DocumentType: "PO"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.summaryCalled {
		t.Fatal("expected summary repository to be called")
	}
	if repo.summaryType != "PO" {
		t.Fatalf("expected repository summary type PO, got %s", repo.summaryType)
	}
	if resp.TotalQuantity != 900 {
		t.Fatalf("expected total quantity 900, got %v", resp.TotalQuantity)
	}
	if resp.DocumentType != "PO" || resp.TotalDocuments != 4 {
		t.Fatalf("unexpected summary response: %+v", resp)
	}
}

func TestGetSummaryReturnsAllTypeDashboardResponse(t *testing.T) {
	repo := &fakeRepository{
		summaryResult: models.SummaryResponse{
			DocumentType: "ALL",
			DN: &models.SummaryMetric{
				TotalDocuments: 3,
				TotalQuantity:  400,
			},
			PO: &models.SummaryMetric{
				TotalDocuments: 3,
				TotalQuantity:  900,
			},
			SO: &models.SummaryMetric{
				TotalDocuments: 2,
				TotalQuantity:  400,
			},
			Total: &models.SummaryMetric{
				TotalDocuments: 8,
				TotalQuantity:  1700,
			},
		},
	}

	svc := New(repo)

	resp, err := svc.GetSummary(context.Background(), models.SummaryRequest{DocumentType: "ALL"})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.summaryType != "ALL" {
		t.Fatalf("expected repository summary type ALL, got %s", repo.summaryType)
	}
	if resp.DocumentType != "ALL" {
		t.Fatalf("expected document type ALL, got %s", resp.DocumentType)
	}
	if resp.DN == nil || resp.PO == nil || resp.SO == nil || resp.Total == nil {
		t.Fatalf("expected dn/po/so/total blocks, got %+v", resp)
	}
	if resp.DN.TotalDocuments != 3 || resp.PO.TotalDocuments != 3 || resp.SO.TotalDocuments != 2 {
		t.Fatalf("unexpected per-type document totals: %+v", resp)
	}
	if resp.DN.TotalQuantity != 400 || resp.PO.TotalQuantity != 900 || resp.SO.TotalQuantity != 400 {
		t.Fatalf("unexpected per-type quantity totals: %+v", resp)
	}
	if resp.Total.TotalDocuments != 8 || resp.Total.TotalQuantity != 1700 {
		t.Fatalf("unexpected grand total block: %+v", resp.Total)
	}
}

func TestGetSummaryReturnsRepositoryError(t *testing.T) {
	repo := &fakeRepository{summaryErr: errors.New("db down")}
	svc := New(repo)

	resp, err := svc.GetSummary(context.Background(), models.SummaryRequest{DocumentType: "DN"})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}
}

func TestUpdateUpdatesDraftDocumentData(t *testing.T) {
	repo := &fakeRepository{
		findByUUIDResult: &models.CustomerOrderDocument{
			UUID:                 "doc-1",
			DocumentType:         "PO",
			DocumentNumber:       "PO-TMC-2026-0001",
			DocumentDate:         time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
			CustomerID:           1,
			CustomerNameSnapshot: "Toyota Old",
			Status:               "draft",
		},
		customerName: "Toyota New",
		itemSnapshots: map[string]*models.ItemSnapshot{
			"LV-009": {PartName: "Steel Plate Z", PartNumber: stringPtr("SP-009"), Model: stringPtr("Camry 2026")},
		},
	}

	svc := New(repo)
	resp, err := svc.Update(context.Background(), "doc-1", models.UpdateOrderRequest{
		CustomerID:      9,
		ContactPerson:   "Budi",
		DeliveryAddress: "Bekasi",
		Notes:           "Urgent",
		Items: []models.UpdateItemInput{{
			ItemUniqCode: "LV-009",
			Quantity:     25,
			DeliveryDate: "2026-04-25",
		}},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if repo.updatedDoc == nil {
		t.Fatal("expected repository update to be called")
	}
	if repo.updatedDoc.CustomerID != 9 || repo.updatedDoc.CustomerNameSnapshot != "Toyota New" {
		t.Fatalf("unexpected updated customer data: %+v", repo.updatedDoc)
	}
	if repo.updatedDoc.ContactPerson == nil || *repo.updatedDoc.ContactPerson != "Budi" {
		t.Fatalf("unexpected contact person: %+v", repo.updatedDoc.ContactPerson)
	}
	if len(repo.updatedDoc.Items) != 1 {
		t.Fatalf("expected 1 updated item, got %d", len(repo.updatedDoc.Items))
	}
	if repo.updatedDoc.Items[0].LineNo != 1 || repo.updatedDoc.Items[0].Quantity != 25 {
		t.Fatalf("unexpected updated item: %+v", repo.updatedDoc.Items[0])
	}
	if resp.TotalQuantity != 25 {
		t.Fatalf("expected total quantity 25, got %v", resp.TotalQuantity)
	}
}

func TestUpdateRejectsNonDraftDocument(t *testing.T) {
	repo := &fakeRepository{
		findByUUIDResult: &models.CustomerOrderDocument{
			UUID:         "doc-1",
			DocumentType: "PO",
			Status:       "active",
		},
	}

	svc := New(repo)
	resp, err := svc.Update(context.Background(), "doc-1", models.UpdateOrderRequest{
		CustomerID: 1,
		Items: []models.UpdateItemInput{{
			ItemUniqCode: "LV-009",
			Quantity:     25,
			DeliveryDate: "2026-04-25",
		}},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if resp != nil {
		t.Fatalf("expected nil response, got %+v", resp)
	}
	if repo.updatedDoc != nil {
		t.Fatal("did not expect repository update to be called")
	}
}

func docCloneForTest(doc *models.CustomerOrderDocument) *models.CustomerOrderDocument {
	if doc == nil {
		return nil
	}
	clone := *doc
	clone.Items = append([]models.CustomerOrderDocumentItem(nil), doc.Items...)
	return &clone
}

func stringPtr(v string) *string { return &v }
