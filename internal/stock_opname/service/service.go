package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	invService "github.com/ganasa18/go-template/internal/inventory/service"
	"github.com/ganasa18/go-template/internal/stock_opname/adjuster"
	stockModels "github.com/ganasa18/go-template/internal/stock_opname/models"
	"github.com/ganasa18/go-template/internal/stock_opname/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/inventoryconst"
	"gorm.io/gorm"
)

type IService interface {
	GetStats(ctx context.Context, inventoryType string) (*stockModels.StockOpnameStats, error)
	ListUniqOptions(ctx context.Context, q stockModels.FormOptionsQuery) ([]stockModels.UniqOption, error)
	GetHistoryLogs(ctx context.Context, q stockModels.HistoryLogsQuery) (*stockModels.HistoryLogListResponse, error)
	GetAuditLogs(ctx context.Context, sessionID int64, page, limit int) (*stockModels.AuditLogListResponse, error)
	ListSessions(ctx context.Context, f repository.SessionFilter) (*stockModels.StockOpnameSessionListResponse, error)
	CreateSession(ctx context.Context, req stockModels.CreateSessionRequest, actor string) (*stockModels.StockOpnameSessionItem, error)
	GetSessionByID(ctx context.Context, id int64) (*stockModels.StockOpnameSessionDetail, error)
	UpdateSession(ctx context.Context, id int64, req stockModels.UpdateSessionRequest, actor string) (*stockModels.StockOpnameSessionItem, error)
	DeleteSession(ctx context.Context, id int64, actor string) error
	AddEntry(ctx context.Context, sessionID int64, req stockModels.CreateEntryRequest, actor string) (*stockModels.StockOpnameEntryItem, error)
	BulkAddEntries(ctx context.Context, sessionID int64, req stockModels.BulkCreateEntryRequest, actor string) (*stockModels.BulkCreateEntryResponse, error)
	UpdateEntry(ctx context.Context, sessionID, entryID int64, req stockModels.UpdateEntryRequest, actor string) (*stockModels.StockOpnameEntryItem, error)
	DeleteEntry(ctx context.Context, sessionID, entryID int64, actor string) error
	SubmitSession(ctx context.Context, id int64, actor string) (*stockModels.StockOpnameSessionItem, error)
	ApproveSession(ctx context.Context, id int64, req stockModels.ApproveRequest, actor string) (*stockModels.StockOpnameSessionItem, error)
	ApproveEntry(ctx context.Context, sessionID, entryID int64, req stockModels.ApproveRequest, actor string) (*stockModels.StockOpnameEntryItem, error)
}

type service struct {
	repo      repository.IRepository
	db        *gorm.DB
	invSvc    invService.IService
	adjusters map[string]adjuster.InventoryAdjuster
}

func New(repo repository.IRepository, db *gorm.DB, invSvc invService.IService) IService {
	return &service{repo: repo, db: db, invSvc: invSvc, adjusters: map[string]adjuster.InventoryAdjuster{
		stockModels.InventoryTypeFG:  adjuster.NewFGAdjuster(),
		stockModels.InventoryTypeRM:  adjuster.NewRMAdjuster(),
		stockModels.InventoryTypeIDR: adjuster.NewIndirectAdjuster(),
		stockModels.InventoryTypeWIP: adjuster.NewWIPAdjuster(),
	}}
}

func (s *service) GetStats(ctx context.Context, inventoryType string) (*stockModels.StockOpnameStats, error) {
	inventoryType = normalizeInventoryType(inventoryType)
	if err := validateInventoryType(inventoryType); err != nil {
		return nil, err
	}
	return s.repo.GetStats(ctx, inventoryType)
}

func (s *service) ListUniqOptions(ctx context.Context, q stockModels.FormOptionsQuery) ([]stockModels.UniqOption, error) {
	adj, err := s.getAdjuster(normalizeInventoryType(q.Type))
	if err != nil {
		return nil, err
	}
	rows, err := adj.SearchUniqs(ctx, s.db, strings.TrimSpace(q.Q), q.Limit)
	if err != nil {
		return nil, err
	}
	items := make([]stockModels.UniqOption, 0, len(rows))
	for i := range rows {
		items = append(items, stockModels.UniqOption{UniqCode: rows[i].UniqCode, PartNumber: rows[i].PartNumber, PartName: rows[i].PartName, UOM: rows[i].UOM, SystemQty: rows[i].SystemQty, WeightKg: rows[i].WeightKg})
	}
	return items, nil
}

func (s *service) GetHistoryLogs(ctx context.Context, q stockModels.HistoryLogsQuery) (*stockModels.HistoryLogListResponse, error) {
	inventoryType := normalizeInventoryType(q.Type)
	if err := validateInventoryType(inventoryType); err != nil {
		return nil, err
	}
	rows, total, err := s.repo.ListHistoryLogs(ctx, inventoryType, strings.TrimSpace(q.UniqCode), q.From, q.To, q.Limit, q.Offset)
	if err != nil {
		return nil, err
	}
	items := make([]stockModels.HistoryLogItem, 0, len(rows))
	for i := range rows {
		item := stockModels.HistoryLogItem(rows[i])
		item.Reason = humanizeHistoryReason(item.Reason)
		items = append(items, item)
	}
	return &stockModels.HistoryLogListResponse{Items: items, Pagination: paginate(total, q.Page, q.Limit)}, nil
}

func (s *service) GetAuditLogs(ctx context.Context, sessionID int64, page, limit int) (*stockModels.AuditLogListResponse, error) {
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 20
	}
	rows, total, err := s.repo.ListAuditLogs(ctx, sessionID, limit, (page-1)*limit)
	if err != nil {
		return nil, err
	}
	items := make([]stockModels.AuditLogItem, 0, len(rows))
	for i := range rows {
		items = append(items, toAuditLogItem(&rows[i]))
	}
	return &stockModels.AuditLogListResponse{Items: items, Pagination: paginate(total, page, limit)}, nil
}

func (s *service) ListSessions(ctx context.Context, f repository.SessionFilter) (*stockModels.StockOpnameSessionListResponse, error) {
	if f.Type != "" {
		f.Type = normalizeInventoryType(f.Type)
		if err := validateInventoryType(f.Type); err != nil {
			return nil, err
		}
	}
	rows, total, err := s.repo.ListSessions(ctx, f)
	if err != nil {
		return nil, err
	}
	items := make([]stockModels.StockOpnameSessionItem, 0, len(rows))
	for i := range rows {
		items = append(items, toSessionListItem(&rows[i]))
	}
	return &stockModels.StockOpnameSessionListResponse{Items: items, Pagination: paginate(total, f.Page, f.Limit)}, nil
}

func (s *service) CreateSession(ctx context.Context, req stockModels.CreateSessionRequest, actor string) (*stockModels.StockOpnameSessionItem, error) {
	req.InventoryType = normalizeInventoryType(req.InventoryType)
	req.Method = normalizeMethod(req.Method)
	if err := validateCreateSessionRequest(req); err != nil {
		return nil, err
	}
	session := &stockModels.StockOpnameSession{InventoryType: req.InventoryType, Method: req.Method, PeriodMonth: req.PeriodMonth, PeriodYear: req.PeriodYear, WarehouseLocation: req.WarehouseLocation, ScheduleDate: parseDate(req.ScheduleDate), CountedDate: parseDate(req.CountedDate), Remarks: trimPtr(req.Remarks), Approver: trimPtr(req.Approver), Status: stockModels.SessionStatusDraft, CreatedBy: strPtr(actor), UpdatedBy: strPtr(actor)}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		number, err := s.repo.GenerateSessionNumber(ctx, tx, req.InventoryType, req.PeriodMonth, req.PeriodYear)
		if err != nil {
			return err
		}
		session.SessionNumber = number
		if err := s.repo.CreateSession(ctx, tx, session); err != nil {
			return err
		}
		if err := s.appendAuditLog(ctx, tx, session.ID, nil, session.InventoryType, stockModels.AuditActionCreate, stockModels.AuditEntitySession, actor, session.Remarks, map[string]interface{}{"session_number": session.SessionNumber, "method": session.Method, "period_month": session.PeriodMonth, "period_year": session.PeriodYear}); err != nil {
			return err
		}
		if len(req.Items) == 0 {
			return nil
		}
		if len(req.Items) > 1 {
			return apperror.UnprocessableEntity("one stock opname row can contain only one item; create separate stock opname rows per uniq")
		}
		adj, err := s.getAdjuster(session.InventoryType)
		if err != nil {
			return err
		}
		entry, err := s.buildEntry(ctx, tx, session, adj, req.Items[0], actor)
		if err != nil {
			return err
		}
		if err := s.repo.CreateEntry(ctx, tx, entry); err != nil {
			return err
		}
		if err := s.appendAuditLog(ctx, tx, session.ID, &entry.ID, session.InventoryType, stockModels.AuditActionAddEntry, stockModels.AuditEntityEntry, actor, entry.Remarks, map[string]interface{}{"uniq_code": entry.UniqCode, "counted_qty": entry.CountedQty, "source": "create_session"}); err != nil {
			return err
		}
		if err := s.repo.RecalculateSessionTotals(ctx, tx, session.ID); err != nil {
			return err
		}
		if err := s.refreshSessionTotals(ctx, tx, session); err != nil {
			return err
		}
		session.Status = stockModels.SessionStatusInProgress
		session.UpdatedBy = strPtr(actor)
		session.UpdatedAt = time.Now()
		if err := s.repo.UpdateSession(ctx, tx, session); err != nil {
			return err
		}
		return s.appendAuditLog(ctx, tx, session.ID, nil, session.InventoryType, stockModels.AuditActionBulkAddEntries, stockModels.AuditEntitySession, actor, session.Remarks, map[string]interface{}{"created": 1, "source": "create_session"})
	})
	if err != nil {
		return nil, err
	}
	item := toSessionItem(session)
	return &item, nil
}

func (s *service) GetSessionByID(ctx context.Context, id int64) (*stockModels.StockOpnameSessionDetail, error) {
	row, err := s.repo.GetSessionWithEntries(ctx, id)
	if err != nil {
		return nil, err
	}
	entries := make([]stockModels.StockOpnameEntryItem, 0, len(row.Entries))
	for i := range row.Entries {
		entries = append(entries, toEntryItem(&row.Entries[i]))
	}
	return &stockModels.StockOpnameSessionDetail{Session: toSessionItem(row), Entries: entries}, nil
}

func (s *service) UpdateSession(ctx context.Context, id int64, req stockModels.UpdateSessionRequest, actor string) (*stockModels.StockOpnameSessionItem, error) {
	var updated *stockModels.StockOpnameSession
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := s.repo.GetSessionByIDTx(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if !isEditableSessionStatus(session.Status) {
			return apperror.Conflict("session can only be updated while draft or in_progress")
		}
		if req.Method != nil {
			m := normalizeMethod(*req.Method)
			if err := validateMethod(m); err != nil {
				return err
			}
			session.Method = m
		}
		if req.PeriodMonth != nil {
			if *req.PeriodMonth < 1 || *req.PeriodMonth > 12 {
				return apperror.UnprocessableEntity("period_month must be between 1 and 12")
			}
			session.PeriodMonth = *req.PeriodMonth
		}
		if req.PeriodYear != nil {
			session.PeriodYear = *req.PeriodYear
		}
		if req.WarehouseLocation != nil {
			session.WarehouseLocation = req.WarehouseLocation
		}
		if req.ScheduleDate != nil {
			session.ScheduleDate = parseDate(req.ScheduleDate)
		}
		if req.CountedDate != nil {
			session.CountedDate = parseDate(req.CountedDate)
		}
		if req.Remarks != nil {
			session.Remarks = trimPtr(req.Remarks)
		}
		if req.Approver != nil {
			session.Approver = trimPtr(req.Approver)
		}
		session.UpdatedBy = strPtr(actor)
		session.UpdatedAt = time.Now()
		updated = session
		if err := s.repo.UpdateSession(ctx, tx, session); err != nil {
			return err
		}
		return s.appendAuditLog(ctx, tx, session.ID, nil, session.InventoryType, stockModels.AuditActionUpdate, stockModels.AuditEntitySession, actor, session.Remarks, map[string]interface{}{"status": session.Status, "method": session.Method})
	})
	if err != nil {
		return nil, err
	}
	item := toSessionItem(updated)
	return &item, nil
}

func (s *service) DeleteSession(ctx context.Context, id int64, actor string) error {
	row, err := s.repo.GetSessionByID(ctx, id)
	if err != nil {
		return err
	}
	if !isEditableSessionStatus(row.Status) {
		return apperror.Conflict("session can only be deleted while draft or in_progress")
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := s.repo.SoftDeleteSession(ctx, tx, id, actor); err != nil {
			return err
		}
		return s.appendAuditLog(ctx, tx, row.ID, nil, row.InventoryType, stockModels.AuditActionDelete, stockModels.AuditEntitySession, actor, row.Remarks, map[string]interface{}{"session_number": row.SessionNumber, "status": row.Status})
	})
	return err
}

func (s *service) AddEntry(ctx context.Context, sessionID int64, req stockModels.CreateEntryRequest, actor string) (*stockModels.StockOpnameEntryItem, error) {
	entry, err := s.addEntryTx(ctx, sessionID, req, actor)
	if err != nil {
		return nil, err
	}
	item := toEntryItem(entry)
	return &item, nil
}

func (s *service) BulkAddEntries(ctx context.Context, sessionID int64, req stockModels.BulkCreateEntryRequest, actor string) (*stockModels.BulkCreateEntryResponse, error) {
	if len(req.Items) == 0 {
		return nil, apperror.BadRequest("items cannot be empty")
	}
	resp := &stockModels.BulkCreateEntryResponse{Errors: []stockModels.BulkCreateEntryError{}}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := s.repo.GetSessionByIDTx(ctx, tx, sessionID, true)
		if err != nil {
			return err
		}
		if !isEditableSessionStatus(session.Status) {
			return apperror.Conflict("entries can only be added while draft or in_progress")
		}
		adj, err := s.getAdjuster(session.InventoryType)
		if err != nil {
			return err
		}
		for i := range req.Items {
			entry, buildErr := s.buildEntry(ctx, tx, session, adj, req.Items[i], actor)
			if buildErr != nil {
				resp.Errors = append(resp.Errors, stockModels.BulkCreateEntryError{Row: i + 1, UniqCode: req.Items[i].UniqCode, Message: buildErr.Error()})
				continue
			}
			if err := s.repo.CreateEntry(ctx, tx, entry); err != nil {
				resp.Errors = append(resp.Errors, stockModels.BulkCreateEntryError{Row: i + 1, UniqCode: req.Items[i].UniqCode, Message: err.Error()})
				continue
			}
			resp.Created++
		}
		if resp.Created > 0 {
			if err := s.repo.RecalculateSessionTotals(ctx, tx, session.ID); err != nil {
				return err
			}
			if err := s.refreshSessionTotals(ctx, tx, session); err != nil {
				return err
			}
			session.Status = stockModels.SessionStatusInProgress
			session.UpdatedBy = strPtr(actor)
			session.UpdatedAt = time.Now()
			if err := s.repo.UpdateSession(ctx, tx, session); err != nil {
				return err
			}
			return s.appendAuditLog(ctx, tx, session.ID, nil, session.InventoryType, stockModels.AuditActionBulkAddEntries, stockModels.AuditEntitySession, actor, nil, map[string]interface{}{"created": resp.Created, "errors": len(resp.Errors)})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (s *service) UpdateEntry(ctx context.Context, sessionID, entryID int64, req stockModels.UpdateEntryRequest, actor string) (*stockModels.StockOpnameEntryItem, error) {
	var updated *stockModels.StockOpnameEntry
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := s.repo.GetSessionByIDTx(ctx, tx, sessionID, true)
		if err != nil {
			return err
		}
		if !isEditableSessionStatus(session.Status) {
			return apperror.Conflict("entries can only be updated while draft or in_progress")
		}
		entry, err := s.repo.GetEntryByIDTx(ctx, tx, entryID, true)
		if err != nil {
			return err
		}
		if entry.SessionID != sessionID {
			return apperror.NotFound("stock opname entry not found for session")
		}
		adj, err := s.getAdjuster(session.InventoryType)
		if err != nil {
			return err
		}
		if req.UniqCode != nil {
			snapshot, err := adj.ResolveUniq(ctx, tx, strings.TrimSpace(*req.UniqCode))
			if err != nil {
				return err
			}
			entry.UniqCode = strings.TrimSpace(*req.UniqCode)
			entry.EntityID = snapshot.EntityID
			entry.PartNumber = snapshot.PartNumber
			entry.PartName = snapshot.PartName
			entry.UOM = snapshot.UOM
			entry.SystemQtySnapshot = snapshot.SystemQty
		}
		if req.CountedQty != nil {
			entry.CountedQty = *req.CountedQty
		}
		if req.WeightKg != nil {
			entry.WeightKg = req.WeightKg
		}
		if req.CyclePengiriman != nil {
			entry.CyclePengiriman = trimPtr(req.CyclePengiriman)
		}
		if req.UserCounter != nil {
			entry.UserCounter = trimPtr(req.UserCounter)
		}
		if req.Remarks != nil {
			entry.Remarks = trimPtr(req.Remarks)
		}
		entry.VariancePct = calcVariancePct(entry.SystemQtySnapshot, entry.CountedQty)
		entry.UpdatedBy = strPtr(actor)
		entry.UpdatedAt = time.Now()
		updated = entry
		if err := s.repo.UpdateEntry(ctx, tx, entry); err != nil {
			return err
		}
		if err := s.repo.RecalculateSessionTotals(ctx, tx, session.ID); err != nil {
			return err
		}
		if err := s.refreshSessionTotals(ctx, tx, session); err != nil {
			return err
		}
		return s.appendAuditLog(ctx, tx, session.ID, &entry.ID, session.InventoryType, stockModels.AuditActionUpdateEntry, stockModels.AuditEntityEntry, actor, entry.Remarks, map[string]interface{}{"uniq_code": entry.UniqCode, "counted_qty": entry.CountedQty})
	})
	if err != nil {
		return nil, err
	}
	item := toEntryItem(updated)
	return &item, nil
}

func (s *service) DeleteEntry(ctx context.Context, sessionID, entryID int64, actor string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := s.repo.GetSessionByIDTx(ctx, tx, sessionID, true)
		if err != nil {
			return err
		}
		if !isEditableSessionStatus(session.Status) {
			return apperror.Conflict("entries can only be deleted while draft or in_progress")
		}
		entry, err := s.repo.GetEntryByIDTx(ctx, tx, entryID, true)
		if err != nil {
			return err
		}
		if entry.SessionID != sessionID {
			return apperror.NotFound("stock opname entry not found for session")
		}
		if err := s.repo.DeleteEntry(ctx, tx, entryID); err != nil {
			return err
		}
		if err := s.repo.RecalculateSessionTotals(ctx, tx, sessionID); err != nil {
			return err
		}
		if err := s.refreshSessionTotals(ctx, tx, session); err != nil {
			return err
		}
		session.UpdatedBy = strPtr(actor)
		session.UpdatedAt = time.Now()
		if err := s.repo.UpdateSession(ctx, tx, session); err != nil {
			return err
		}
		return s.appendAuditLog(ctx, tx, session.ID, &entry.ID, session.InventoryType, stockModels.AuditActionDeleteEntry, stockModels.AuditEntityEntry, actor, entry.Remarks, map[string]interface{}{"uniq_code": entry.UniqCode})
	})
}

func (s *service) SubmitSession(ctx context.Context, id int64, actor string) (*stockModels.StockOpnameSessionItem, error) {
	var updated *stockModels.StockOpnameSession
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := s.repo.GetSessionByIDTx(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if session.Status != stockModels.SessionStatusDraft && session.Status != stockModels.SessionStatusInProgress {
			return apperror.Conflict("session can only be submitted from draft or in_progress")
		}
		entries, err := s.repo.ListEntriesBySessionTx(ctx, tx, session.ID)
		if err != nil {
			return err
		}
		if len(entries) <= 0 {
			return apperror.Conflict("session must have at least one entry before submit")
		}
		session.TotalEntries = len(entries)
		now := time.Now()
		session.Status = stockModels.SessionStatusPendingApproval
		session.SubmittedBy = strPtr(actor)
		session.SubmittedAt = &now
		session.UpdatedBy = strPtr(actor)
		session.UpdatedAt = now
		updated = session
		if err := s.repo.UpdateSession(ctx, tx, session); err != nil {
			return err
		}
		return s.appendAuditLog(ctx, tx, session.ID, nil, session.InventoryType, stockModels.AuditActionSubmit, stockModels.AuditEntitySession, actor, session.Remarks, map[string]interface{}{"status": session.Status, "total_entries": session.TotalEntries})
	})
	if err != nil {
		return nil, err
	}
	item := toSessionItem(updated)
	return &item, nil
}

func (s *service) ApproveSession(ctx context.Context, id int64, req stockModels.ApproveRequest, actor string) (*stockModels.StockOpnameSessionItem, error) {
	var updated *stockModels.StockOpnameSession
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := s.repo.GetSessionByIDTx(ctx, tx, id, true)
		if err != nil {
			return err
		}
		if session.Status == stockModels.SessionStatusApproved || session.Status == stockModels.SessionStatusRejected || session.Status == stockModels.SessionStatusPartiallyApproved {
			return apperror.Conflict("session already finished")
		}
		action := normalizeAction(req.Action)
		if err := validateAction(action); err != nil {
			return err
		}
		entries, err := s.repo.ListEntriesBySessionTx(ctx, tx, session.ID)
		if err != nil {
			return err
		}
		if len(entries) <= 0 {
			return apperror.Conflict("session has no entry to approve")
		}
		session.TotalEntries = len(entries)
		adj, err := s.getAdjuster(session.InventoryType)
		if err != nil {
			return err
		}
		now := time.Now()
		for i := range entries {
			if entries[i].Status != stockModels.EntryStatusPending {
				continue
			}
			if action == stockModels.ApprovalActionApprove {
				result, err := adj.ApplyAdjustment(ctx, tx, &entries[i], session.SessionNumber, actor)
				if err != nil {
					return err
				}
				if err := s.appendInventoryLog(ctx, tx, session.InventoryType, &entries[i], actor, session.SessionNumber, result); err != nil {
					return err
				}
				entries[i].Status = stockModels.EntryStatusApproved
				entries[i].RejectReason = nil
			} else {
				entries[i].Status = stockModels.EntryStatusRejected
				entries[i].RejectReason = trimPtr(req.Remarks)
			}
			entries[i].ApprovedBy = strPtr(actor)
			entries[i].ApprovedAt = &now
			entries[i].UpdatedBy = strPtr(actor)
			entries[i].UpdatedAt = now
			if err := s.repo.UpdateEntry(ctx, tx, &entries[i]); err != nil {
				return err
			}
		}
		status, err := s.repo.DeriveSessionStatus(ctx, tx, session.ID)
		if err != nil {
			return err
		}
		session.Status = status
		session.ApprovedBy = strPtr(actor)
		session.ApprovedAt = &now
		session.ApprovalRemarks = trimPtr(req.Remarks)
		session.UpdatedBy = strPtr(actor)
		session.UpdatedAt = now
		updated = session
		if err := s.repo.UpdateSession(ctx, tx, session); err != nil {
			return err
		}
		auditAction := stockModels.AuditActionApproveSession
		if action == stockModels.ApprovalActionReject {
			auditAction = stockModels.AuditActionRejectSession
		}
		return s.appendAuditLog(ctx, tx, session.ID, nil, session.InventoryType, auditAction, stockModels.AuditEntitySession, actor, trimPtr(req.Remarks), map[string]interface{}{"status": session.Status, "action": action})
	})
	if err != nil {
		return nil, err
	}
	item := toSessionItem(updated)
	return &item, nil
}

func (s *service) ApproveEntry(ctx context.Context, sessionID, entryID int64, req stockModels.ApproveRequest, actor string) (*stockModels.StockOpnameEntryItem, error) {
	var updated *stockModels.StockOpnameEntry
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := s.repo.GetSessionByIDTx(ctx, tx, sessionID, true)
		if err != nil {
			return err
		}
		if session.Status == stockModels.SessionStatusApproved || session.Status == stockModels.SessionStatusRejected || session.Status == stockModels.SessionStatusPartiallyApproved {
			return apperror.Conflict("session already finished")
		}
		entry, err := s.repo.GetEntryByIDTx(ctx, tx, entryID, true)
		if err != nil {
			return err
		}
		if entry.SessionID != sessionID {
			return apperror.NotFound("stock opname entry not found for session")
		}
		if entry.Status != stockModels.EntryStatusPending {
			return apperror.Conflict("entry already processed")
		}
		action := normalizeAction(req.Action)
		if err := validateAction(action); err != nil {
			return err
		}
		adj, err := s.getAdjuster(session.InventoryType)
		if err != nil {
			return err
		}
		now := time.Now()
		if action == stockModels.ApprovalActionApprove {
			result, err := adj.ApplyAdjustment(ctx, tx, entry, session.SessionNumber, actor)
			if err != nil {
				return err
			}
			if err := s.appendInventoryLog(ctx, tx, session.InventoryType, entry, actor, session.SessionNumber, result); err != nil {
				return err
			}
			entry.Status = stockModels.EntryStatusApproved
			entry.RejectReason = nil
		} else {
			entry.Status = stockModels.EntryStatusRejected
			entry.RejectReason = trimPtr(req.Remarks)
		}
		entry.ApprovedBy = strPtr(actor)
		entry.ApprovedAt = &now
		entry.UpdatedBy = strPtr(actor)
		entry.UpdatedAt = now
		updated = entry
		if err := s.repo.UpdateEntry(ctx, tx, entry); err != nil {
			return err
		}
		status, err := s.repo.DeriveSessionStatus(ctx, tx, session.ID)
		if err != nil {
			return err
		}
		session.Status = status
		session.ApprovedBy = strPtr(actor)
		session.ApprovedAt = &now
		session.ApprovalRemarks = trimPtr(req.Remarks)
		session.UpdatedBy = strPtr(actor)
		session.UpdatedAt = now
		if err := s.repo.UpdateSession(ctx, tx, session); err != nil {
			return err
		}
		auditAction := stockModels.AuditActionApproveEntry
		if action == stockModels.ApprovalActionReject {
			auditAction = stockModels.AuditActionRejectEntry
		}
		return s.appendAuditLog(ctx, tx, session.ID, &entry.ID, session.InventoryType, auditAction, stockModels.AuditEntityEntry, actor, trimPtr(req.Remarks), map[string]interface{}{"entry_status": entry.Status, "session_status": session.Status, "uniq_code": entry.UniqCode})
	})
	if err != nil {
		return nil, err
	}
	item := toEntryItem(updated)
	return &item, nil
}

func (s *service) addEntryTx(ctx context.Context, sessionID int64, req stockModels.CreateEntryRequest, actor string) (*stockModels.StockOpnameEntry, error) {
	var created *stockModels.StockOpnameEntry
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		session, err := s.repo.GetSessionByIDTx(ctx, tx, sessionID, true)
		if err != nil {
			return err
		}
		if !isEditableSessionStatus(session.Status) {
			return apperror.Conflict("entries can only be added while draft or in_progress")
		}
		adj, err := s.getAdjuster(session.InventoryType)
		if err != nil {
			return err
		}
		entry, err := s.buildEntry(ctx, tx, session, adj, req, actor)
		if err != nil {
			return err
		}
		created = entry
		if err := s.repo.CreateEntry(ctx, tx, entry); err != nil {
			return err
		}
		if err := s.repo.RecalculateSessionTotals(ctx, tx, session.ID); err != nil {
			return err
		}
		if err := s.refreshSessionTotals(ctx, tx, session); err != nil {
			return err
		}
		session.Status = stockModels.SessionStatusInProgress
		session.UpdatedBy = strPtr(actor)
		session.UpdatedAt = time.Now()
		if err := s.repo.UpdateSession(ctx, tx, session); err != nil {
			return err
		}
		return s.appendAuditLog(ctx, tx, session.ID, &entry.ID, session.InventoryType, stockModels.AuditActionAddEntry, stockModels.AuditEntityEntry, actor, entry.Remarks, map[string]interface{}{"uniq_code": entry.UniqCode, "counted_qty": entry.CountedQty})
	})
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (s *service) buildEntry(ctx context.Context, tx *gorm.DB, session *stockModels.StockOpnameSession, adj adjuster.InventoryAdjuster, req stockModels.CreateEntryRequest, actor string) (*stockModels.StockOpnameEntry, error) {
	uniqCode := strings.TrimSpace(req.UniqCode)
	if uniqCode == "" {
		return nil, apperror.BadRequest("uniq_code is required")
	}
	snapshot, err := adj.ResolveUniq(ctx, tx, uniqCode)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	entry := &stockModels.StockOpnameEntry{SessionID: session.ID, UniqCode: uniqCode, EntityID: snapshot.EntityID, PartNumber: snapshot.PartNumber, PartName: snapshot.PartName, UOM: snapshot.UOM, SystemQtySnapshot: snapshot.SystemQty, CountedQty: req.CountedQty, VariancePct: calcVariancePct(snapshot.SystemQty, req.CountedQty), WeightKg: req.WeightKg, CyclePengiriman: trimPtr(req.CyclePengiriman), UserCounter: trimPtr(req.UserCounter), Remarks: trimPtr(req.Remarks), Status: stockModels.EntryStatusPending, CreatedBy: strPtr(actor), UpdatedBy: strPtr(actor), CreatedAt: now, UpdatedAt: now}
	return entry, nil
}

func (s *service) appendInventoryLog(ctx context.Context, tx *gorm.DB, inventoryType string, entry *stockModels.StockOpnameEntry, actor, sessionNumber string, result *adjuster.AdjustmentResult) error {
	notes := fmt.Sprintf("stock opname %s", sessionNumber)
	if result == nil {
		return nil
	}
	switch inventoryType {
	case stockModels.InventoryTypeRM:
		if err := s.invSvc.AppendMovementLog(ctx, tx, invService.MovementLogInput{Category: string(inventoryconst.CategoryRawMaterial), MovementType: string(inventoryconst.MovementStockOpname), UniqCode: entry.UniqCode, EntityID: entry.EntityID, QtyChange: result.QtyChange, WeightChange: result.WeightChange, SourceFlag: string(inventoryconst.SourceStockOpname), ReferenceID: &sessionNumber, Notes: &notes, LoggedBy: actor}); err != nil {
			return apperror.Internal("append RM stock opname movement log: " + err.Error())
		}
	case stockModels.InventoryTypeIDR:
		if err := s.invSvc.AppendMovementLog(ctx, tx, invService.MovementLogInput{Category: string(inventoryconst.CategoryIndirectMaterial), MovementType: string(inventoryconst.MovementStockOpname), UniqCode: entry.UniqCode, EntityID: entry.EntityID, QtyChange: result.QtyChange, WeightChange: result.WeightChange, SourceFlag: string(inventoryconst.SourceStockOpname), ReferenceID: &sessionNumber, Notes: &notes, LoggedBy: actor}); err != nil {
			return apperror.Internal("append indirect stock opname movement log: " + err.Error())
		}
	}
	return nil
}

func (s *service) appendAuditLog(ctx context.Context, tx *gorm.DB, sessionID int64, entryID *int64, inventoryType, action, entityType, actor string, remarks *string, metadata map[string]interface{}) error {
	log := &stockModels.StockOpnameAuditLog{SessionID: sessionID, EntryID: entryID, InventoryType: inventoryType, Action: action, EntityType: entityType, Actor: actor, Remarks: remarks, Metadata: repository.ToJSONMap(metadata)}
	return s.repo.CreateAuditLog(ctx, tx, log)
}

func (s *service) refreshSessionTotals(ctx context.Context, tx *gorm.DB, session *stockModels.StockOpnameSession) error {
	refreshed, err := s.repo.GetSessionByIDTx(ctx, tx, session.ID, false)
	if err != nil {
		return err
	}
	session.TotalEntries = refreshed.TotalEntries
	session.TotalVarianceQty = refreshed.TotalVarianceQty
	return nil
}

func (s *service) getAdjuster(inventoryType string) (adjuster.InventoryAdjuster, error) {
	adj, ok := s.adjusters[inventoryType]
	if !ok {
		return nil, apperror.UnprocessableEntity("inventory_type must be one of: FG, RM, IDR, WIP")
	}
	return adj, nil
}

func toSessionItem(row *stockModels.StockOpnameSession) stockModels.StockOpnameSessionItem {
	statusLabel, impactLabel := deriveStatusLabels(row.Status)
	periodLabel := fmt.Sprintf("%02d/%d", row.PeriodMonth, row.PeriodYear)
	return stockModels.StockOpnameSessionItem{ID: row.ID, UUID: row.UUID.String(), SessionNumber: row.SessionNumber, InventoryType: row.InventoryType, Method: row.Method, PeriodMonth: row.PeriodMonth, PeriodYear: row.PeriodYear, PeriodLabel: periodLabel, WarehouseLocation: row.WarehouseLocation, ScheduleDate: row.ScheduleDate, CountedDate: row.CountedDate, Remarks: row.Remarks, TotalEntries: row.TotalEntries, TotalVarianceQty: row.TotalVarianceQty, Status: row.Status, StatusLabel: statusLabel, ImpactLabel: impactLabel, SubmittedBy: row.SubmittedBy, SubmittedAt: row.SubmittedAt, Approver: row.Approver, ApprovedBy: row.ApprovedBy, ApprovedAt: row.ApprovedAt, ApprovalRemarks: row.ApprovalRemarks, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func toSessionListItem(row *repository.SessionListRow) stockModels.StockOpnameSessionItem {
	statusLabel, impactLabel := deriveStatusLabels(row.Status)
	periodLabel := fmt.Sprintf("%02d/%d", row.PeriodMonth, row.PeriodYear)
	return stockModels.StockOpnameSessionItem{ID: row.ID, UUID: row.UUID.String(), SessionNumber: row.SessionNumber, InventoryType: row.InventoryType, Method: row.Method, PeriodMonth: row.PeriodMonth, PeriodYear: row.PeriodYear, PeriodLabel: periodLabel, WarehouseLocation: row.WarehouseLocation, ScheduleDate: row.ScheduleDate, CountedDate: row.CountedDate, Remarks: row.Remarks, TotalEntries: row.TotalEntries, TotalVarianceQty: row.TotalVarianceQty, SystemQtyTotal: row.SystemQtyTotal, PhysicalQtyTotal: row.PhysicalQtyTotal, VarianceQtyTotal: row.VarianceQtyTotal, VariancePctTotal: row.VariancePctTotal, CostImpact: row.CostImpact, Status: row.Status, StatusLabel: statusLabel, ImpactLabel: impactLabel, SubmittedBy: row.SubmittedBy, SubmittedAt: row.SubmittedAt, Approver: row.Approver, ApprovedBy: row.ApprovedBy, ApprovedAt: row.ApprovedAt, ApprovalRemarks: row.ApprovalRemarks, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func toEntryItem(row *stockModels.StockOpnameEntry) stockModels.StockOpnameEntryItem {
	return stockModels.StockOpnameEntryItem{ID: row.ID, UUID: row.UUID.String(), SessionID: row.SessionID, UniqCode: row.UniqCode, EntityID: row.EntityID, PartNumber: row.PartNumber, PartName: row.PartName, UOM: row.UOM, SystemQtySnapshot: row.SystemQtySnapshot, CountedQty: row.CountedQty, VarianceQty: row.CountedQty - row.SystemQtySnapshot, VariancePct: row.VariancePct, WeightKg: row.WeightKg, CyclePengiriman: row.CyclePengiriman, UserCounter: row.UserCounter, Remarks: row.Remarks, Status: row.Status, ApprovedBy: row.ApprovedBy, ApprovedAt: row.ApprovedAt, RejectReason: row.RejectReason, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt}
}

func toAuditLogItem(row *stockModels.StockOpnameAuditLog) stockModels.AuditLogItem {
	var metadata interface{}
	if len(row.Metadata) > 0 {
		_ = json.Unmarshal(row.Metadata, &metadata)
	}
	return stockModels.AuditLogItem{ID: row.ID, UUID: row.UUID.String(), SessionID: row.SessionID, EntryID: row.EntryID, InventoryType: row.InventoryType, Action: row.Action, EntityType: row.EntityType, Actor: row.Actor, Remarks: row.Remarks, Metadata: metadata, CreatedAt: row.CreatedAt}
}

func deriveStatusLabels(status string) (string, string) {
	switch status {
	case stockModels.SessionStatusApproved:
		return "Completed", "Approved"
	case stockModels.SessionStatusRejected:
		return "Completed", "Rejected"
	case stockModels.SessionStatusPartiallyApproved:
		return "Completed", "Partially Approved"
	case stockModels.SessionStatusPendingApproval:
		return "Pending Verification", "Waiting for Approval"
	case stockModels.SessionStatusInProgress:
		return "In Progress", "Counting In Progress"
	default:
		return "Draft", "Not Submitted"
	}
}

func humanizeHistoryReason(reason *string) *string {
	if reason == nil {
		return nil
	}
	v := strings.TrimSpace(*reason)
	if v == "" {
		return nil
	}
	switch v {
	case "stock_opname":
		v = "Stock Opname"
	case "incoming":
		v = "Incoming"
	case "outgoing":
		v = "Outgoing"
	case "adjustment":
		v = "Adjustment"
	}
	return &v
}

func paginate(total int64, page, limit int) stockModels.Pagination {
	totalPages := 1
	if limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
		if totalPages == 0 {
			totalPages = 1
		}
	}
	if page < 1 {
		page = 1
	}
	return stockModels.Pagination{Total: total, Page: page, Limit: limit, TotalPages: totalPages}
}

func normalizeInventoryType(v string) string { return strings.ToUpper(strings.TrimSpace(v)) }
func normalizeMethod(v string) string        { return strings.ToLower(strings.TrimSpace(v)) }
func normalizeAction(v string) string        { return strings.ToLower(strings.TrimSpace(v)) }
func strPtr(v string) *string                { return &v }

func trimPtr(v *string) *string {
	if v == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*v)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseDate(v *string) *time.Time {
	if v == nil || strings.TrimSpace(*v) == "" {
		return nil
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(*v))
	if err != nil {
		return nil
	}
	return &t
}

func calcVariancePct(systemQty, countedQty float64) *float64 {
	if systemQty == 0 {
		return nil
	}
	v := ((countedQty - systemQty) / systemQty) * 100
	return &v
}

func isEditableSessionStatus(status string) bool {
	return status == stockModels.SessionStatusDraft || status == stockModels.SessionStatusInProgress
}

func validateInventoryType(v string) error {
	if _, ok := stockModels.ValidInventoryTypes[v]; !ok {
		return apperror.UnprocessableEntity("inventory_type must be one of: FG, RM, IDR, WIP")
	}
	return nil
}

func validateMethod(v string) error {
	if v != stockModels.MethodManual && v != stockModels.MethodBulk {
		return apperror.UnprocessableEntity("method must be one of: manual, bulk")
	}
	return nil
}

func validateAction(v string) error {
	if v != stockModels.ApprovalActionApprove && v != stockModels.ApprovalActionReject {
		return apperror.UnprocessableEntity("action must be one of: approve, reject")
	}
	return nil
}

func validateCreateSessionRequest(req stockModels.CreateSessionRequest) error {
	if err := validateInventoryType(req.InventoryType); err != nil {
		return err
	}
	if err := validateMethod(req.Method); err != nil {
		return err
	}
	if req.PeriodMonth < 1 || req.PeriodMonth > 12 {
		return apperror.UnprocessableEntity("period_month must be between 1 and 12")
	}
	if req.PeriodYear < 2000 {
		return apperror.UnprocessableEntity("period_year is invalid")
	}
	for i := range req.Items {
		if strings.TrimSpace(req.Items[i].UniqCode) == "" {
			return apperror.UnprocessableEntity(fmt.Sprintf("items[%d].uniq_code is required", i))
		}
	}
	return nil
}
