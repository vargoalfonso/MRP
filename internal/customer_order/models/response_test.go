package models

import (
	"testing"
	"time"
)

func TestToDocumentResponseComputesDetailSummaryForDN(t *testing.T) {
	documentDate := time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC)
	firstDeliveryDate := time.Date(2026, 4, 20, 0, 0, 0, 0, time.UTC)
	secondDeliveryDate := time.Date(2026, 4, 21, 0, 0, 0, 0, time.UTC)
	model := "Camry 2024"

	doc := &CustomerOrderDocument{
		UUID:                 "doc-1",
		DocumentType:         "DN",
		DocumentNumber:       "DN-TMC-2026-0001",
		DocumentDate:         documentDate,
		PeriodSchedule:       "April 2026",
		CustomerID:           1,
		CustomerNameSnapshot: "Toyota",
		Status:               "active",
		Items: []CustomerOrderDocumentItem{
			{
				UUID:         "item-1",
				LineNo:       1,
				ItemUniqCode: "LV-001",
				PartName:     "Steel Plate",
				PartNumber:   "SP-001-A",
				Model:        &model,
				Quantity:     120,
				DeliveryDate: &firstDeliveryDate,
			},
			{
				UUID:         "item-2",
				LineNo:       2,
				ItemUniqCode: "LV-001",
				PartName:     "Steel Plate X",
				PartNumber:   "SP-001-B",
				Model:        &model,
				Quantity:     80,
				DeliveryDate: &secondDeliveryDate,
			},
			{
				UUID:         "item-3",
				LineNo:       3,
				ItemUniqCode: "LV-002",
				PartName:     "Steel Plate Y",
				PartNumber:   "SP-001-C",
				Model:        &model,
				Quantity:     50,
			},
		},
	}

	resp := ToDocumentResponse(doc)

	if resp.TotalQuantity != 250 {
		t.Fatalf("expected total quantity 250, got %v", resp.TotalQuantity)
	}
	if resp.TotalUniq != 2 {
		t.Fatalf("expected total uniq 2, got %d", resp.TotalUniq)
	}
	if resp.HeaderDeliveryDate == nil {
		t.Fatal("expected header delivery date for DN")
	}
	if !resp.HeaderDeliveryDate.Equal(documentDate) {
		t.Fatalf("expected header delivery date %v, got %v", documentDate, resp.HeaderDeliveryDate)
	}
}

func TestToDocumentResponseLeavesHeaderDeliveryDateNilForPO(t *testing.T) {
	doc := &CustomerOrderDocument{
		UUID:                 "doc-2",
		DocumentType:         "PO",
		DocumentNumber:       "PO-TMC-2026-0001",
		DocumentDate:         time.Date(2026, 4, 19, 0, 0, 0, 0, time.UTC),
		CustomerID:           1,
		CustomerNameSnapshot: "Toyota",
		Status:               "draft",
		Items: []CustomerOrderDocumentItem{
			{UUID: "item-1", LineNo: 1, ItemUniqCode: "LV-001", PartName: "Steel Plate", PartNumber: "SP-001-A", Quantity: 100},
		},
	}

	resp := ToDocumentResponse(doc)

	if resp.HeaderDeliveryDate != nil {
		t.Fatalf("expected nil header delivery date for PO, got %v", resp.HeaderDeliveryDate)
	}
}
