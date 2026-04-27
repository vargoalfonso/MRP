package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	awmodels "github.com/ganasa18/go-template/internal/approval_workflow/models"
	"github.com/ganasa18/go-template/internal/prl/models"
	prlRepo "github.com/ganasa18/go-template/internal/prl/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/approval"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

var forecastPeriodPattern = regexp.MustCompile(`^\d{4}-Q[1-4]$`)

type Service interface {
	CreateUniqBOM(ctx context.Context, req models.CreateUniqBOMRequest) (*models.UniqBillOfMaterial, error)
	GetUniqBOMByUUID(ctx context.Context, uuid string) (*models.UniqBillOfMaterial, error)
	ListUniqBOMs(ctx context.Context, query models.ListUniqBOMQuery) (*models.UniqBOMListResult, error)
	UpdateUniqBOM(ctx context.Context, uuid string, req models.UpdateUniqBOMRequest) (*models.UniqBillOfMaterial, error)
	DeleteUniqBOM(ctx context.Context, uuid string) error

	CreatePRL(ctx context.Context, req models.CreatePRLRequest, submittedBy string) (*models.PRL, error)
	BulkCreatePRLs(ctx context.Context, req models.BulkCreatePRLRequest, submittedBy string) (*models.BulkCreatePRLResponse, error)
	GetPRLByUUID(ctx context.Context, uuid string) (*models.PRL, error)
	GetPRLDetail(ctx context.Context, id string) (*models.PRLDetailResponse, error)
	ListPRLs(ctx context.Context, query models.ListPRLQuery) (*models.PRLListResult, error)
	UpdatePRL(ctx context.Context, uuid string, req models.UpdatePRLRequest) (*models.PRL, error)
	DeletePRL(ctx context.Context, uuid string) error
	ApprovePRLs(ctx context.Context, req models.BulkStatusActionRequest, userID string, userRoles []string) (*models.BulkStatusActionResponse, error)
	RejectPRLs(ctx context.Context, req models.BulkStatusActionRequest, userID string, userRoles []string) (*models.BulkStatusActionResponse, error)

	ListCustomerLookups(ctx context.Context, search string) ([]models.CustomerLookup, error)
	ListForecastPeriodOptions(year int) []models.ForecastPeriodOption
	ImportPRLs(ctx context.Context, fileName string, reader io.Reader, submittedBy string) (*models.ImportPRLResponse, error)
	ExportPRLs(ctx context.Context, query models.ListPRLQuery) (string, []byte, error)
}

type service struct {
	repo prlRepo.IRepository
	db   *gorm.DB
}

func New(repo prlRepo.IRepository, db *gorm.DB) Service {
	return &service{repo: repo, db: db}
}

func (s *service) CreateUniqBOM(ctx context.Context, req models.CreateUniqBOMRequest) (*models.UniqBillOfMaterial, error) {
	item := &models.UniqBillOfMaterial{
		UUID:         uuid.NewString(),
		UniqCode:     strings.ToUpper(models.Trimmed(req.UniqCode)),
		ProductModel: models.Trimmed(req.ProductModel),
		PartName:     models.Trimmed(req.PartName),
		PartNumber:   strings.ToUpper(models.Trimmed(req.PartNumber)),
	}
	if err := s.repo.CreateUniqBOM(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *service) GetUniqBOMByUUID(ctx context.Context, uuid string) (*models.UniqBillOfMaterial, error) {
	if models.Trimmed(uuid) == "" {
		return nil, apperror.BadRequest("uniq bom id is required")
	}
	return s.repo.FindUniqBOMByUUID(ctx, uuid)
}

func (s *service) ListUniqBOMs(ctx context.Context, query models.ListUniqBOMQuery) (*models.UniqBOMListResult, error) {
	page, limit := normalizePageLimit(query.Page, query.Limit)
	filters := models.UniqBOMListFilters{
		Search: models.Trimmed(query.Search),
		Page:   page,
		Limit:  limit,
		Offset: (page - 1) * limit,
	}
	items, total, err := s.repo.ListUniqBOMs(ctx, filters)
	if err != nil {
		return nil, err
	}
	return &models.UniqBOMListResult{Items: items, Pagination: models.NewPaginationMeta(page, limit, total)}, nil
}

func (s *service) UpdateUniqBOM(ctx context.Context, uuid string, req models.UpdateUniqBOMRequest) (*models.UniqBillOfMaterial, error) {
	item, err := s.GetUniqBOMByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	item.UniqCode = strings.ToUpper(models.Trimmed(req.UniqCode))
	item.ProductModel = models.Trimmed(req.ProductModel)
	item.PartName = models.Trimmed(req.PartName)
	item.PartNumber = strings.ToUpper(models.Trimmed(req.PartNumber))
	if err := s.repo.UpdateUniqBOM(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *service) DeleteUniqBOM(ctx context.Context, uuid string) error {
	item, err := s.GetUniqBOMByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	return s.repo.DeleteUniqBOM(ctx, item)
}

func (s *service) CreatePRL(ctx context.Context, req models.CreatePRLRequest, submittedBy string) (*models.PRL, error) {
	submittedBy = strings.TrimSpace(submittedBy)
	if submittedBy == "" {
		return nil, apperror.Unauthorized("not authenticated")
	}

	period, err := normalizeForecastPeriod(req.ForecastPeriod)
	if err != nil {
		return nil, err
	}
	if req.Quantity < 1 {
		return nil, apperror.BadRequest("quantity must be greater than 0")
	}

	customer, err := s.resolveCustomer(ctx, req.CustomerUUID, req.CustomerCode)
	if err != nil {
		return nil, err
	}

	uniqCode := strings.ToUpper(models.Trimmed(req.UniqCode))
	bom, err := s.repo.FindUniqBOMByUniqCode(ctx, uniqCode)
	if err != nil {
		if appErr, ok := apperror.As(err); ok && appErr.Code == apperror.CodeNotFound {
			productModel := models.Trimmed(req.ProductModel)
			partName := models.Trimmed(req.PartName)
			partNumber := strings.ToUpper(models.Trimmed(req.PartNumber))
			if productModel == "" {
				return nil, apperror.BadRequest("product_model is required when uniq_code is new")
			}
			if partName == "" {
				return nil, apperror.BadRequest("part_name is required when uniq_code is new")
			}
			if partNumber == "" {
				return nil, apperror.BadRequest("part_number is required when uniq_code is new")
			}

			newBOM := &models.UniqBillOfMaterial{
				UUID:         uuid.NewString(),
				UniqCode:     uniqCode,
				ProductModel: productModel,
				PartName:     partName,
				PartNumber:   partNumber,
			}
			if createErr := s.repo.CreateUniqBOM(ctx, newBOM); createErr != nil {
				return nil, createErr
			}
			bom = newBOM
		} else {
			return nil, err
		}
	}

	// Optional mismatch guard: if client supplies BOM fields, they must match the existing UNIQ BOM.
	if models.Trimmed(req.ProductModel) != "" && models.Trimmed(req.ProductModel) != bom.ProductModel {
		return nil, apperror.BadRequest("product_model does not match existing uniq bom")
	}
	if models.Trimmed(req.PartName) != "" && models.Trimmed(req.PartName) != bom.PartName {
		return nil, apperror.BadRequest("part_name does not match existing uniq bom")
	}
	if models.Trimmed(req.PartNumber) != "" && strings.ToUpper(models.Trimmed(req.PartNumber)) != bom.PartNumber {
		return nil, apperror.BadRequest("part_number does not match existing uniq bom")
	}

	item := &models.PRL{
		UUID:           uuid.NewString(),
		PRLID:          "PENDING",
		CustomerUUID:   customer.UUID,
		CustomerCode:   customer.CustomerID,
		CustomerName:   customer.CustomerName,
		UniqBOMUUID:    bom.UUID,
		UniqCode:       bom.UniqCode,
		ProductModel:   bom.ProductModel,
		PartName:       bom.PartName,
		PartNumber:     bom.PartNumber,
		ForecastPeriod: period,
		Quantity:       req.Quantity,
		Status:         models.PRLStatusPending,
	}

	if err := s.repo.CreatePRLs(ctx, []*models.PRL{item}); err != nil {
		return nil, err
	}
	log.Println("PRL ID:", item.ID)
	if err := s.createApprovalInstance(ctx, item.ID, submittedBy); err != nil {
		return nil, err
	}

	return item, nil
}

func (s *service) BulkCreatePRLs(ctx context.Context, req models.BulkCreatePRLRequest, submittedBy string) (*models.BulkCreatePRLResponse, error) {
	if len(req.Entries) == 0 {
		return nil, apperror.BadRequest("entries is required")
	}
	submittedBy = strings.TrimSpace(submittedBy)
	if submittedBy == "" {
		return nil, apperror.Unauthorized("not authenticated")
	}

	items := make([]*models.PRL, 0, len(req.Entries))
	for index, entry := range req.Entries {
		item, err := s.buildPRLFromEntry(ctx, entry)
		if err != nil {
			return nil, apperror.BadRequest(fmt.Sprintf("entry %d: %s", index+1, err.Error()))
		}
		items = append(items, item)
	}

	if err := s.repo.CreatePRLs(ctx, items); err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := s.createApprovalInstance(ctx, item.ID, submittedBy); err != nil {
			return nil, err
		}
	}

	out := make([]models.PRL, 0, len(items))
	for _, item := range items {
		out = append(out, *item)
	}

	return &models.BulkCreatePRLResponse{CreatedCount: len(out), Items: out}, nil
}

func (s *service) GetPRLByUUID(ctx context.Context, uuid string) (*models.PRL, error) {
	if models.Trimmed(uuid) == "" {
		return nil, apperror.BadRequest("prl id is required")
	}
	return s.repo.FindPRLByUUID(ctx, uuid)
}

func (s *service) GetPRLDetail(ctx context.Context, id string) (*models.PRLDetailResponse, error) {
	item, err := s.resolvePRLDetailTarget(ctx, id)
	if err != nil {
		return nil, err
	}

	resp := &models.PRLDetailResponse{PRL: *item}

	var instance awmodels.ApprovalInstance
	if err := s.db.WithContext(ctx).
		Where("action_name = ? AND reference_table = ? AND reference_id = ?", "prl", "prls", item.ID).
		First(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return resp, nil
		}
		return nil, apperror.InternalWrap("find approval instance failed", err)
	}

	var workflow awmodels.ApprovalWorkflow
	if err := s.db.WithContext(ctx).Where("id = ?", instance.ApprovalWorkflowID).First(&workflow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			resp.Approval = &models.PRLApprovalSummary{
				InstanceID:       instance.ID,
				WorkflowID:       instance.ApprovalWorkflowID,
				CurrentLevel:     instance.CurrentLevel,
				MaxLevel:         instance.MaxLevel,
				Status:           instance.Status,
				SubmittedBy:      instance.SubmittedBy,
				ApprovalProgress: instance.ApprovalProgress,
			}
			return resp, nil
		}
		return nil, apperror.InternalWrap("find approval workflow failed", err)
	}

	resp.Approval = &models.PRLApprovalSummary{
		InstanceID:       instance.ID,
		WorkflowID:       workflow.ID,
		WorkflowAction:   workflow.ActionName,
		CurrentLevel:     instance.CurrentLevel,
		MaxLevel:         instance.MaxLevel,
		Status:           instance.Status,
		SubmittedBy:      instance.SubmittedBy,
		ApprovalProgress: instance.ApprovalProgress,
		LevelRoles: models.PRLApprovalWorkflowRoles{
			Level1: workflow.Level1Role,
			Level2: workflow.Level2Role,
			Level3: workflow.Level3Role,
			Level4: workflow.Level4Role,
		},
	}

	return resp, nil
}

func (s *service) resolvePRLDetailTarget(ctx context.Context, id string) (*models.PRL, error) {
	trimmed := models.Trimmed(id)
	if trimmed == "" {
		return nil, apperror.BadRequest("prl id is required")
	}
	if numericID, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		return s.repo.FindPRLByID(ctx, numericID)
	}
	return s.repo.FindPRLByUUID(ctx, trimmed)
}

func (s *service) ListPRLs(ctx context.Context, query models.ListPRLQuery) (*models.PRLListResult, error) {
	page, limit := normalizePageLimit(query.Page, query.Limit)
	status, err := normalizeOptionalStatus(query.Status)
	if err != nil {
		return nil, err
	}
	period, err := normalizeOptionalForecastPeriod(query.ForecastPeriod)
	if err != nil {
		return nil, err
	}
	customerUUID := normalizeOptionalString(query.CustomerUUID)
	uniqCode := normalizeOptionalUpper(query.UniqCode)

	filters := models.PRLListFilters{
		Search:         models.Trimmed(query.Search),
		Status:         status,
		ForecastPeriod: period,
		CustomerUUID:   customerUUID,
		UniqCode:       uniqCode,
		Page:           page,
		Limit:          limit,
		Offset:         (page - 1) * limit,
	}
	items, total, err := s.repo.ListPRLs(ctx, filters)
	if err != nil {
		return nil, err
	}
	return &models.PRLListResult{Items: items, Pagination: models.NewPaginationMeta(page, limit, total)}, nil
}

func (s *service) UpdatePRL(ctx context.Context, uuid string, req models.UpdatePRLRequest) (*models.PRL, error) {
	item, err := s.GetPRLByUUID(ctx, uuid)
	if err != nil {
		return nil, err
	}
	period, err := normalizeForecastPeriod(req.ForecastPeriod)
	if err != nil {
		return nil, err
	}
	if req.Quantity < 1 {
		return nil, apperror.BadRequest("quantity must be greater than 0")
	}
	item.ForecastPeriod = period
	item.Quantity = req.Quantity
	if err := s.repo.UpdatePRL(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *service) DeletePRL(ctx context.Context, uuid string) error {
	item, err := s.GetPRLByUUID(ctx, uuid)
	if err != nil {
		return err
	}
	return s.repo.DeletePRL(ctx, item)
}

func (s *service) ApprovePRLs(ctx context.Context, req models.BulkStatusActionRequest, userID string, userRoles []string) (*models.BulkStatusActionResponse, error) {
	return s.bulkProcessApprovalAction(ctx, req, userID, userRoles, models.PRLStatusApproved)
}

func (s *service) RejectPRLs(ctx context.Context, req models.BulkStatusActionRequest, userID string, userRoles []string) (*models.BulkStatusActionResponse, error) {
	return s.bulkProcessApprovalAction(ctx, req, userID, userRoles, models.PRLStatusRejected)
}

func (s *service) ListCustomerLookups(ctx context.Context, search string) ([]models.CustomerLookup, error) {
	return s.repo.ListCustomers(ctx, search)
}

func (s *service) ListForecastPeriodOptions(year int) []models.ForecastPeriodOption {
	if year <= 0 {
		year = time.Now().Year()
	}
	items := make([]models.ForecastPeriodOption, 0, 4)
	for quarter := 1; quarter <= 4; quarter++ {
		value := fmt.Sprintf("%d-Q%d", year, quarter)
		items = append(items, models.ForecastPeriodOption{Value: value, Label: value})
	}
	return items
}

func (s *service) ImportPRLs(ctx context.Context, fileName string, reader io.Reader, submittedBy string) (*models.ImportPRLResponse, error) {
	if !strings.EqualFold(filepath.Ext(fileName), ".xlsx") {
		return nil, apperror.BadRequest("file must be .xlsx")
	}

	workbook, err := excelize.OpenReader(reader)
	if err != nil {
		return nil, apperror.BadRequest("failed to read excel file")
	}
	defer func() { _ = workbook.Close() }()

	sheetName := workbook.GetSheetName(0)
	if sheetName == "" {
		return nil, apperror.BadRequest("excel file has no sheet")
	}

	rows, err := workbook.GetRows(sheetName)
	if err != nil {
		return nil, apperror.BadRequest("failed to parse excel rows")
	}
	if len(rows) < 2 {
		return nil, apperror.BadRequest("excel file must contain header and at least one data row")
	}

	entries := make([]models.CreatePRLEntryRequest, 0, len(rows)-1)
	for index, row := range rows[1:] {
		if isRowEmpty(row) {
			continue
		}
		if len(row) < 4 {
			return nil, apperror.BadRequest(fmt.Sprintf("row %d: expected 4 columns", index+2))
		}
		quantity, convErr := strconv.ParseInt(strings.TrimSpace(row[3]), 10, 64)
		if convErr != nil {
			return nil, apperror.BadRequest(fmt.Sprintf("row %d: quantity must be a number", index+2))
		}
		entries = append(entries, models.CreatePRLEntryRequest{
			CustomerCode:   strings.TrimSpace(row[0]),
			UniqCode:       strings.TrimSpace(row[1]),
			ForecastPeriod: strings.TrimSpace(row[2]),
			Quantity:       quantity,
		})
	}

	resp, err := s.BulkCreatePRLs(ctx, models.BulkCreatePRLRequest{Entries: entries}, submittedBy)
	if err != nil {
		return nil, err
	}

	return &models.ImportPRLResponse{ImportedCount: resp.CreatedCount, Items: resp.Items}, nil
}

func (s *service) ExportPRLs(ctx context.Context, query models.ListPRLQuery) (string, []byte, error) {
	status, err := normalizeOptionalStatus(query.Status)
	if err != nil {
		return "", nil, err
	}
	period, err := normalizeOptionalForecastPeriod(query.ForecastPeriod)
	if err != nil {
		return "", nil, err
	}
	filters := models.PRLListFilters{
		Search:         models.Trimmed(query.Search),
		Status:         status,
		ForecastPeriod: period,
		CustomerUUID:   normalizeOptionalString(query.CustomerUUID),
		UniqCode:       normalizeOptionalUpper(query.UniqCode),
	}

	items, err := s.repo.ListPRLsForExport(ctx, filters)
	if err != nil {
		return "", nil, err
	}

	file := excelize.NewFile()
	sheetName := file.GetSheetName(0)
	headers := []string{"PRL ID", "Customer ID", "Customer Name", "UNIQ Code", "Product Model", "Part Name", "Part Number", "Forecast Period", "Quantity", "Status", "Created At"}
	for index, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(index+1, 1)
		_ = file.SetCellValue(sheetName, cell, header)
	}

	for rowIndex, item := range items {
		row := rowIndex + 2
		values := []interface{}{item.PRLID, item.CustomerCode, item.CustomerName, item.UniqCode, item.ProductModel, item.PartName, item.PartNumber, item.ForecastPeriod, item.Quantity, item.Status, item.CreatedAt.Format(time.RFC3339)}
		for colIndex, value := range values {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, row)
			_ = file.SetCellValue(sheetName, cell, value)
		}
	}

	buffer, err := file.WriteToBuffer()
	if err != nil {
		return "", nil, apperror.InternalWrap("generate export excel failed", err)
	}

	filename := fmt.Sprintf("prl-export-%s.xlsx", time.Now().Format("20060102-150405"))
	return filename, buffer.Bytes(), nil
}

func (s *service) buildPRLFromEntry(ctx context.Context, entry models.CreatePRLEntryRequest) (*models.PRL, error) {
	period, err := normalizeForecastPeriod(entry.ForecastPeriod)
	if err != nil {
		return nil, err
	}
	if entry.Quantity < 1 {
		return nil, apperror.BadRequest("quantity must be greater than 0")
	}

	customer, err := s.resolveCustomer(ctx, entry.CustomerUUID, entry.CustomerCode)
	if err != nil {
		return nil, err
	}
	bom, err := s.repo.FindUniqBOMByUniqCode(ctx, strings.ToUpper(models.Trimmed(entry.UniqCode)))
	if err != nil {
		return nil, err
	}

	return &models.PRL{
		UUID:           uuid.NewString(),
		PRLID:          "PENDING",
		CustomerUUID:   customer.UUID,
		CustomerCode:   customer.CustomerID,
		CustomerName:   customer.CustomerName,
		UniqBOMUUID:    bom.UUID,
		UniqCode:       bom.UniqCode,
		ProductModel:   bom.ProductModel,
		PartName:       bom.PartName,
		PartNumber:     bom.PartNumber,
		ForecastPeriod: period,
		Quantity:       entry.Quantity,
		Status:         models.PRLStatusPending,
	}, nil
}

func (s *service) resolveCustomer(ctx context.Context, customerUUID models.CustomerUUIDInput, customerCode string) (*struct {
	UUID         string
	CustomerID   string
	CustomerName string
}, error) {
	if (!customerUUID.IsInt && strings.TrimSpace(customerUUID.UUID) == "") && (!customerUUID.IsInt || customerUUID.RowID <= 0) && strings.TrimSpace(customerCode) == "" {
		return nil, apperror.BadRequest("customer_uuid or customer_code is required")
	}

	if customerUUID.IsInt {
		if customerUUID.RowID <= 0 {
			return nil, apperror.BadRequest("customer_uuid must be greater than 0")
		}
		customer, err := s.repo.FindCustomerByRowID(ctx, customerUUID.RowID)
		if err != nil {
			return nil, err
		}
		return &struct {
			UUID         string
			CustomerID   string
			CustomerName string
		}{UUID: customer.UUID, CustomerID: customer.CustomerID, CustomerName: customer.CustomerName}, nil
	}

	if strings.TrimSpace(customerUUID.UUID) != "" {
		customer, err := s.repo.FindCustomerByUUID(ctx, customerUUID.UUID)
		if err != nil {
			return nil, err
		}
		return &struct {
			UUID         string
			CustomerID   string
			CustomerName string
		}{UUID: customer.UUID, CustomerID: customer.CustomerID, CustomerName: customer.CustomerName}, nil
	}

	customer, err := s.repo.FindCustomerByCode(ctx, strings.ToUpper(models.Trimmed(customerCode)))
	if err != nil {
		return nil, err
	}
	return &struct {
		UUID         string
		CustomerID   string
		CustomerName string
	}{UUID: customer.UUID, CustomerID: customer.CustomerID, CustomerName: customer.CustomerName}, nil
}

func (s *service) createApprovalInstance(ctx context.Context, referenceID int64, submittedBy string) error {
	_, err := approval.CreateInstance(ctx, s.db, approval.CreateInstanceParams{
		ActionName:     "prl",
		ReferenceTable: "prls",
		ReferenceID:    referenceID,
		SubmittedBy:    submittedBy,
	})
	return err
}

func (s *service) bulkProcessApprovalAction(ctx context.Context, req models.BulkStatusActionRequest, userID string, userRoles []string, action string) (*models.BulkStatusActionResponse, error) {
	ids, err := normalizeActionIDs(req.IDs)
	if err != nil {
		return nil, err
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, apperror.Unauthorized("not authenticated")
	}
	roles := normalizeRoles(userRoles)
	if len(roles) == 0 {
		return nil, apperror.Forbidden("no roles assigned for approval")
	}

	results := make([]models.BulkStatusActionResult, 0, len(ids))
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, id := range ids {
			result, processErr := s.processPRLApprovalAction(ctx, tx, id, req.Note, userID, roles, action)
			if processErr != nil {
				return processErr
			}
			results = append(results, *result)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &models.BulkStatusActionResponse{
		UpdatedCount: int64(len(results)),
		Status:       action,
		Results:      results,
	}, nil
}

func (s *service) processPRLApprovalAction(ctx context.Context, tx *gorm.DB, prlID, note, userID string, userRoles []string, action string) (*models.BulkStatusActionResult, error) {
	item, err := s.findPRLByAnyID(ctx, tx, prlID)
	if err != nil {
		return nil, err
	}

	var instance awmodels.ApprovalInstance
	if err := tx.WithContext(ctx).
		Where("action_name = ? AND reference_table = ? AND reference_id = ?", "prl", "prls", item.ID).
		First(&instance).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.BadRequest("approval instance for prl not found")
		}
		return nil, apperror.InternalWrap("find approval instance failed", err)
	}

	var workflow awmodels.ApprovalWorkflow
	if err := tx.WithContext(ctx).
		Where("id = ? AND status = ?", instance.ApprovalWorkflowID, "active").
		First(&workflow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.BadRequest("active approval workflow for prl not found")
		}
		return nil, apperror.InternalWrap("find approval workflow failed", err)
	}

	if instance.Status == models.PRLStatusApproved {
		return nil, apperror.BadRequest("prl already approved")
	}
	if instance.Status == models.PRLStatusRejected {
		return nil, apperror.BadRequest("prl already rejected")
	}
	if instance.CurrentLevel < 1 || instance.CurrentLevel > len(instance.ApprovalProgress.Levels) {
		return nil, apperror.BadRequest("invalid approval current level")
	}

	levelIdx := instance.CurrentLevel - 1
	level := &instance.ApprovalProgress.Levels[levelIdx]
	requiredRole := approval.LevelRole(&workflow, int16(instance.CurrentLevel))
	if requiredRole == "" {
		requiredRole = level.Role
	}
	if requiredRole == "" {
		return nil, apperror.BadRequest("required approval role is not configured")
	}
	if !approval.HasRole(userRoles, requiredRole) {
		return nil, apperror.Forbidden(fmt.Sprintf("prl is waiting approval from role '%s'", requiredRole))
	}
	if level.Status != models.PRLStatusPending {
		return nil, apperror.BadRequest("current approval level is no longer pending")
	}

	now := time.Now().UTC()
	level.Note = strings.TrimSpace(note)
	level.ApprovedBy = userID
	level.ApprovedAt = now.Format(time.RFC3339)

	prlUpdates := map[string]interface{}{
		"updated_at":  now,
		"approved_at": nil,
		"rejected_at": nil,
	}

	if action == models.PRLStatusApproved {
		level.Status = models.PRLStatusApproved
		if instance.CurrentLevel >= instance.MaxLevel {
			instance.Status = models.PRLStatusApproved
			prlUpdates["status"] = models.PRLStatusApproved
			prlUpdates["approved_at"] = now
		} else {
			instance.CurrentLevel++
			instance.Status = models.PRLStatusPending
			prlUpdates["status"] = models.PRLStatusPending
		}
	} else {
		level.Status = models.PRLStatusRejected
		instance.Status = models.PRLStatusRejected
		prlUpdates["status"] = models.PRLStatusRejected
		prlUpdates["rejected_at"] = now
	}

	instance.UpdatedAt = now
	if err := tx.WithContext(ctx).
		Model(&awmodels.ApprovalInstance{}).
		Where("id = ?", instance.ID).
		Updates(map[string]interface{}{
			"approval_progress": instance.ApprovalProgress,
			"status":            instance.Status,
			"current_level":     instance.CurrentLevel,
			"updated_at":        now,
		}).Error; err != nil {
		return nil, apperror.InternalWrap("update approval instance failed", err)
	}

	if err := tx.WithContext(ctx).
		Model(&models.PRL{}).
		Where("id = ?", item.ID).
		Updates(prlUpdates).Error; err != nil {
		return nil, apperror.InternalWrap("update prl status failed", err)
	}

	return &models.BulkStatusActionResult{ID: item.UUID, CurrentLevel: instance.CurrentLevel, Status: instance.Status}, nil
}

func (s *service) findPRLByAnyID(ctx context.Context, tx *gorm.DB, id string) (*models.PRL, error) {
	trimmed := models.Trimmed(id)
	if trimmed == "" {
		return nil, apperror.BadRequest("prl id is required")
	}

	var item models.PRL
	query := tx.WithContext(ctx)
	if numericID, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
		if err := query.Where("id = ?", numericID).First(&item).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return nil, apperror.NotFound("prl not found")
			}
			return nil, apperror.InternalWrap("find prl failed", err)
		}
		return &item, nil
	}

	if err := query.Where("uuid = ?", trimmed).First(&item).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("prl not found")
		}
		return nil, apperror.InternalWrap("find prl failed", err)
	}
	return &item, nil
}

func normalizeActionIDs(rawIDs []string) ([]string, error) {
	if len(rawIDs) == 0 {
		return nil, apperror.BadRequest("ids is required")
	}
	ids := make([]string, 0, len(rawIDs))
	for _, id := range rawIDs {
		trimmed := models.Trimmed(id)
		if trimmed == "" {
			continue
		}
		ids = append(ids, trimmed)
	}
	if len(ids) == 0 {
		return nil, apperror.BadRequest("ids is required")
	}
	return ids, nil
}

func normalizeRoles(userRoles []string) []string {
	roles := make([]string, 0, len(userRoles))
	for _, role := range userRoles {
		trimmed := strings.TrimSpace(role)
		if trimmed == "" {
			continue
		}
		roles = append(roles, trimmed)
	}
	return roles
}

func normalizePageLimit(page, limit int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}
	return page, limit
}

func normalizeForecastPeriod(value string) (string, error) {
	// Free-text: accept any non-empty value and store it as-is.
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", apperror.BadRequest("forecast_period is required")
	}
	return trimmed, nil
}

func normalizeOptionalForecastPeriod(value string) (*string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	period, err := normalizeForecastPeriod(value)
	if err != nil {
		return nil, err
	}
	return &period, nil
}

func normalizeOptionalStatus(value string) (*string, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	status := strings.ToLower(strings.TrimSpace(value))
	switch status {
	case models.PRLStatusPending, models.PRLStatusApproved, models.PRLStatusRejected:
		return &status, nil
	default:
		return nil, apperror.BadRequest("status must be one of: pending, approved, rejected")
	}
}

func normalizeOptionalString(value string) *string {
	trimmed := models.Trimmed(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func normalizeOptionalUpper(value string) *string {
	trimmed := models.Trimmed(value)
	if trimmed == "" {
		return nil
	}
	upper := strings.ToUpper(trimmed)
	return &upper
}

func isRowEmpty(row []string) bool {
	for _, column := range row {
		if strings.TrimSpace(column) != "" {
			return false
		}
	}
	return true
}

func BuildSampleImportFile() ([]byte, error) {
	file := excelize.NewFile()
	sheet := file.GetSheetName(0)
	headers := []string{"customer_code", "uniq_code", "forecast_period", "quantity"}
	for index, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(index+1, 1)
		_ = file.SetCellValue(sheet, cell, header)
	}
	rows := [][]interface{}{{"CUST-2026-001", "LV7-001", "2026-Q1", 1200}, {"CUST-2026-001", "EM-001-LV7", "2026-Q2", 950}}
	for rowIndex, row := range rows {
		for colIndex, value := range row {
			cell, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+2)
			_ = file.SetCellValue(sheet, cell, value)
		}
	}
	buffer := bytes.NewBuffer(nil)
	if err := file.Write(buffer); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
