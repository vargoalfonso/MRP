// Package service implements business logic for the Bill of Material module.
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"

	awmodels "github.com/ganasa18/go-template/internal/approval_workflow/models"
	"github.com/ganasa18/go-template/internal/billmaterial/models"
	"github.com/ganasa18/go-template/internal/billmaterial/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/ganasa18/go-template/pkg/approval"
	"github.com/ganasa18/go-template/pkg/bulkimport"
	"github.com/ganasa18/go-template/pkg/concurrency"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/google/uuid"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type IService interface {
	// List — expandable BOM tree (parent rows, children loaded per parent)
	ListBom(ctx context.Context, q models.ListBomQuery) (*models.ListBomResponse, error)

	// Create — wizard: parent + routing + material spec + nested children in one call
	CreateBom(ctx context.Context, req models.CreateBomRequest) (*models.BomDetailResponse, error)

	// Detail — full tree with process routes and material spec
	GetBomDetail(ctx context.Context, bomID int64) (*models.BomDetailResponse, error)
	GetBomDetailByVersion(ctx context.Context, bomID int64, version int) (*models.BomDetailResponse, error)
	GetBomVersions(ctx context.Context, bomID int64) (*models.BomVersionsResponse, error)
	CreateBomRevision(ctx context.Context, bomID int64, req models.CreateBomRevisionRequest) (*models.CreateBomRevisionResponse, error)
	// ActivateBomVersion sets a specific BOM version as current without creating a new version.
	ActivateBomVersion(ctx context.Context, bomID int64) (*models.BomDetailResponse, error)

	AddProcessRoute(ctx context.Context, bomID int64, req []models.AddProcessRouteRequest) ([]models.ProcessRouteMutationResponse, error)
	PatchProcessRoute(ctx context.Context, bomID, routeID int64, req models.PatchProcessRouteRequest) (*models.ProcessRouteMutationResponse, error)
	// Update BOM header and parent item fields (partial update)
	UpdateBom(ctx context.Context, bomID int64, req models.UpdateBomRequest) (*models.BomDetailResponse, error)

	// Update a child node (BomLine + its underlying Item)
	UpdateBomChild(ctx context.Context, bomID, lineID int64, req models.UpdateBomChildRequest) (*models.BomDetailResponse, error)

	// Delete parent BOM header (children lines are removed by cascade)
	DeleteBom(ctx context.Context, bomID int64) error

	// Delete a child subtree from BOM by child item id
	DeleteBomChild(ctx context.Context, bomID, childItemID int64) (int64, error)

	// Delete a subtree from BOM by line id (frontend-friendly unique node target)
	DeleteBomLine(ctx context.Context, bomID, lineID int64) (int64, error)

	// ApproveBom processes an approve or reject action for the BOM's approval instance.
	// userRoles are the caller's JWT roles used to verify the current-level role.
	// On full approval, bom_item.status and all child items.status are set to Active.
	ApproveBom(ctx context.Context, bomID int64, userID string, userRoles []string, req models.ApproveBomRequest) (*awmodels.ApprovalInstance, error)

	// Import BOM from Excel, download template, and download generated error file.
	DownloadImportTemplate(ctx context.Context) ([]byte, error)
	ImportFromExcel(ctx context.Context, filePath, fileName, uploadedBy, requestID string) (bulkimport.BulkResult, error)
	DownloadImportErrors(ctx context.Context, token string) ([]byte, error)
	ListImportHistory(ctx context.Context, limit int) ([]models.BomImportHistory, error)
	DownloadImportHistoryError(ctx context.Context, id string) ([]byte, error)

	//full snapshot fetch + atomic clone-edit-submit replace
	GetBomFull(ctx context.Context, bomID int64) (*models.BomFullResponse, error)
	// ReplaceBom clones the BOM with all edits from req, uploads new asset files in parallel,
	// deactivates the old version (is_current=false), and opens a new approval instance.
	// files maps upload_key → multipart file header (may be empty when no files changed).
	ReplaceBom(ctx context.Context, bomID int64, req models.ReplaceBomRequest, files map[string]*multipart.FileHeader, userID string) (*models.ReplaceBomResponse, error)
}

type service struct {
	repo       repository.IRepository
	errorStore bulkimport.ErrorStore
}

type lineTreeKey struct {
	parentItemID int64
	level        int16
}

type bomPreload struct {
	items           map[int64]models.Item
	latestRevisions map[int64]models.ItemRevision
	revisionByID    map[int64]models.ItemRevision
	assets          map[int64]models.ItemAsset
	specs           map[int64]models.ItemMaterialSpec
	routesByRevID   map[int64][]models.ProcessRouteDetail
	children        map[lineTreeKey][]models.BomLine
}

func New(repo repository.IRepository, store bulkimport.ErrorStore) IService {
	if store == nil {
		s, err := bulkimport.NewFileStore("")
		if err == nil {
			store = s
		}
	}
	return &service{repo: repo, errorStore: store}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func (s *service) ListBom(ctx context.Context, q models.ListBomQuery) (*models.ListBomResponse, error) {
	// Normalise so Meta always reflects the values actually used
	limit := q.Limit
	if limit < 1 || limit > 200 {
		limit = 20
	}
	page := q.Page
	if page < 1 {
		page = 1
	}

	bomItems, total, err := s.repo.ListBomItems(ctx, repository.ListFilter{
		UniqCode:            q.UniqCode,
		Status:              q.Status,
		Search:              q.Search,
		SupplierID:          q.SupplierID,
		TypeMaterial:        q.TypeMaterial,
		ExcludeSupplierUUID: q.ExcludeSupplierUUID,
		Page:                page,
		Limit:               limit,
		OrderBy:             q.OrderBy,
		OrderDirection:      q.OrderDirection,
	})
	if err != nil {
		return nil, err
	}

	bomIDs := make([]int64, 0, len(bomItems))
	for _, item := range bomItems {
		bomIDs = append(bomIDs, item.ID)
	}

	lines, err := s.repo.GetBomLinesByBomIDs(ctx, bomIDs)
	if err != nil {
		return nil, err
	}

	preload, err := s.preloadBomData(ctx, bomItems, lines)
	if err != nil {
		return nil, err
	}

	linesByBomID := make(map[int64][]models.BomLine)
	for _, line := range lines {
		linesByBomID[line.BomItemID] = append(linesByBomID[line.BomItemID], line)
	}

	rows := make([]models.BomTreeRow, 0, len(bomItems))

	for _, b := range bomItems {
		parent, ok := preload.items[b.ItemID]
		if !ok {
			continue
		}

		// Build parent row
		bomID := b.ID
		row := models.BomTreeRow{
			ID:         parent.ID,
			BomID:      &bomID,
			UniqCode:   parent.UniqCode,
			PartName:   parent.PartName,
			PartNumber: parent.PartNumber,
			Model:      parent.Model,
			Uom:        parent.Uom,
			Level:      "Parent",
			Asset:      s.buildAssetInfo(preload.assetByItemID(parent.ID)),
			Status:     parent.Status,
		}
		if parentRev, ok := preload.revisionForParent(b); ok {
			row.Version = &parentRev.Revision
			if spec, ok := preload.specs[parentRev.ID]; ok {
				row.MaterialSpec = s.toSpecDetail(&spec)
			}
			// [wo-estimated-time] kirim juga process route (berisi machine_id).
			if routes, ok := preload.routesByRevID[parentRev.ID]; ok {
				row.ProcessRoutes = routes
			}
		}

		row.Children = s.buildChildTree(linesByBomID[b.ID], preload, parent.ID, 1, q.TypeMaterial)

		rows = append(rows, row)
	}

	return &models.ListBomResponse{
		Pagination: pagination.NewMetaBom(total, pagination.BomPaginationInput{
			Page:  page,
			Limit: limit,
		}),
		Items: rows,
	}, nil
}

// buildChildTree recursively builds child rows at a given level from flat lines.
// typeMaterialFilter: when non-empty, only children whose spec.TypeMaterial matches are included.
func (s *service) buildChildTree(lines []models.BomLine, preload *bomPreload, parentItemID int64, level int16, typeMaterialFilter string) []models.BomTreeRow {
	children := preload.childrenByParent(parentItemID, level, lines)
	rows := make([]models.BomTreeRow, 0, len(children))
	for _, line := range children {
		child, ok := preload.items[line.ChildItemID]
		if !ok {
			continue
		}

		qpu := line.QtyPerUniq
		row := models.BomTreeRow{
			ID:         child.ID,
			LineID:     &line.ID,
			UniqCode:   child.UniqCode,
			PartName:   child.PartName,
			PartNumber: child.PartNumber,
			Model:      child.Model,
			Uom:        child.Uom,
			Level:      int(level),
			QPU:        &qpu,
			Asset:      s.buildAssetInfo(preload.assetByItemID(child.ID)),
			Status:     child.Status,
		}
		if rev, ok := preload.revisionForChild(line, child.ID); ok {
			row.Version = &rev.Revision
			// [wo-estimated-time] child juga butuh process route untuk kapasitas mesin.
			if routes, ok := preload.routesByRevID[rev.ID]; ok {
				row.ProcessRoutes = routes
			}
			if spec, ok := preload.specs[rev.ID]; ok {
				row.MaterialSpec = s.toSpecDetail(&spec)
				if typeMaterialFilter != "" {
					if spec.TypeMaterial == nil || *spec.TypeMaterial != typeMaterialFilter {
						continue
					}
				}
			} else if typeMaterialFilter != "" {
				continue
			}
		} else if typeMaterialFilter != "" {
			continue
		}
		if level < 6 {
			row.Children = s.buildChildTree(lines, preload, child.ID, level+1, typeMaterialFilter)
		}
		rows = append(rows, row)
	}
	return rows
}

// buildAssetInfo converts an ItemAsset (or nil) into the AssetInfo response struct.
// asset_type mapping:
//
//	"3d-model" → cad_viewable: true,  label: "3D Available"
//	"drawing"  → cad_viewable: false, label: "2D Available"
//	"photo"    → cad_viewable: false, label: "2D Available"
//	nil        → cad_viewable: false, label: "-"
func (s *service) buildAssetInfo(asset *models.ItemAsset) models.AssetInfo {
	if asset == nil {
		return models.AssetInfo{Label: "-"}
	}
	info := models.AssetInfo{
		ID:        &asset.ID,
		URL:       &asset.FileURL,
		AssetType: asset.AssetType,
	}
	if asset.AssetType == "3d-model" {
		info.Label = "3D Available"
		info.CadViewable = true
	} else {
		info.Label = "2D Available"
		info.CadViewable = false
	}
	return info
}

// ---------------------------------------------------------------------------
// Create (wizard — one call)
// ---------------------------------------------------------------------------

func (s *service) CreateBom(ctx context.Context, req models.CreateBomRequest) (*models.BomDetailResponse, error) {
	// Pre-fetch read-only config outside the transaction to fail fast.
	wf, err := s.repo.GetApprovalWorkflowByActionName(ctx, "bom")
	if err != nil {
		return nil, err
	}
	if wf == nil {
		return nil, apperror.BadRequest("no active approval workflow configured for action 'bom'")
	}
	maxLevel := approval.MaxLevel(wf)
	if maxLevel < 1 {
		return nil, apperror.BadRequest("no approval levels configured for workflow 'bom'")
	}

	var result *models.BomDetailResponse
	txErr := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txSvc := &service{repo: repository.New(tx), errorStore: s.errorStore}

		// 1. Create parent item
		// Semua items (parent & child) mulai sebagai Draft — baru jadi Active
		// setelah BOM selesai di-approve di semua level.
		parent := &models.Item{
			UniqCode:   req.UniqCode,
			PartName:   req.PartName,
			PartNumber: req.PartNumber,
			Model:      req.Model,
			Uom:        req.Uom,
			Status:     "Draft",
		}
		if err := txSvc.repo.CreateItem(ctx, parent); err != nil {
			return err
		}

		// 2. Create default revision
		revStr := "v1"
		parent.CurrentRevision = &revStr
		_ = txSvc.repo.UpdateItem(ctx, parent)
		rev := &models.ItemRevision{
			ItemID:     parent.ID,
			Revision:   revStr,
			Status:     "Draft",
			ChangeNote: req.Description,
		}
		if err := txSvc.repo.CreateRevision(ctx, rev); err != nil {
			return err
		}

		// 3. Picture
		if req.PictureURL != nil {
			_ = txSvc.repo.CreateAsset(ctx, &models.ItemAsset{
				ItemID:    parent.ID,
				AssetType: "photo",
				FileURL:   *req.PictureURL,
				Status:    "Active",
			})
		}

		// 4. Process routes
		if len(req.ProcessRoutes) > 0 {
			if err := txSvc.createRouting(ctx, parent.ID, rev.ID, req.ProcessRoutes); err != nil {
				return err
			}
		}

		// 5. Material spec
		if req.MaterialSpec != nil {
			if err := txSvc.saveMaterialSpec(ctx, rev.ID, req.MaterialSpec); err != nil {
				return err
			}
		}

		// 6. BOM header
		bom := &models.BomItem{
			ItemID:             parent.ID,
			RootItemRevisionID: &rev.ID,
			Version:            1,
			Status:             "Released",
			Description:        req.Description,
			ChangeNote:         req.Description,
			IsCurrent:          true,
		}
		if err := txSvc.repo.CreateBomItem(ctx, bom); err != nil {
			return err
		}

		// 6a. Auto-create approval instance
		instance := &awmodels.ApprovalInstance{
			ActionName:         "bom",
			ReferenceTable:     "bom_item",
			ReferenceID:        bom.ID,
			ApprovalWorkflowID: wf.ID,
			CurrentLevel:       1,
			MaxLevel:           maxLevel,
			Status:             "pending",
			ApprovalProgress:   approval.BuildProgress(wf, maxLevel),
		}
		if err := txSvc.repo.CreateApprovalInstance(ctx, instance); err != nil {
			return err
		}

		// 7. Recurse children
		if err := txSvc.createChildren(ctx, bom.ID, parent.ID, req.Children); err != nil {
			return err
		}

		detail, err := txSvc.GetBomDetail(ctx, bom.ID)
		if err != nil {
			return err
		}
		result = detail
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}
	return result, nil
}

// createChildren resolves or creates each child item and the bom_line, then recurses.
func (s *service) createChildren(ctx context.Context, bomID, parentItemID int64, children []models.ChildInput) error {
	for _, c := range children {
		childID, childRevID, err := s.resolveOrCreateItem(ctx, c)
		if err != nil {
			return err
		}
		// NOTE: Guard "child cannot be the same as parent" SENGAJA DILEPAS.
		// Child part boleh memakai uniq_code yang sama dengan parent BoM
		// (mis. sebuah part multi-fungsi yang juga menjadi salah satu
		// komponen dirinya sendiri di tingkat sub-assembly, atau sekadar
		// re-use item existing). resolveOrCreateItem sudah me-reuse item
		// existing, sehingga tidak menimbulkan duplikat di tabel items.
		// Berlaku untuk create, edit, dan bulk import.

		line := &models.BomLine{
			BomItemID:           bomID,
			ParentItemID:        parentItemID,
			ChildItemID:         childID,
			ChildItemRevisionID: childRevID,
			Level:               c.Level,
			QtyPerUniq:          c.QtyPerUniq,
		}
		if c.ScrapFactor != nil {
			line.ScrapFactor = *c.ScrapFactor
		}
		if c.IsPhantom != nil {
			line.IsPhantom = *c.IsPhantom
		}
		if err := s.repo.CreateBomLine(ctx, line); err != nil {
			return err
		}

		if len(c.Children) > 0 && c.Level < 6 {
			if err := s.createChildren(ctx, bomID, childID, c.Children); err != nil {
				return err
			}
		}
	}
	return nil
}

// resolveOrCreateItem returns the item ID, creating a new item if needed.
func (s *service) resolveOrCreateItem(ctx context.Context, c models.ChildInput) (int64, *int64, error) {
	if c.ItemID != nil {
		if _, err := s.repo.GetItemByID(ctx, *c.ItemID); err != nil {
			return 0, nil, err
		}
		rev, err := s.repo.GetLatestRevision(ctx, *c.ItemID)
		if err != nil {
			return 0, nil, err
		}
		if rev == nil {
			return *c.ItemID, nil, nil
		}
		return *c.ItemID, &rev.ID, nil
	}

	if c.UniqCode == nil || c.PartName == nil {
		return 0, nil, apperror.BadRequest("child must have item_id or uniq_code + part_name")
	}

	// Child UNIQ tidak perlu unik antar BoM: jika item dengan uniq_code yang sama
	// sudah ada di database, reuse item tersebut (termasuk untuk parent BoM lain).
	// Cek existing HARUS dilakukan sebelum guard UOM agar user tidak dipaksa
	// mengirim UOM ulang ketika child yang direferensikan memang sudah ada.
	if existing, err := s.repo.GetItemByUniq(ctx, *c.UniqCode); err == nil && existing != nil {
		rev, _ := s.repo.GetLatestRevision(ctx, existing.ID)
		if rev != nil {
			return existing.ID, &rev.ID, nil
		}
		return existing.ID, nil, nil
	}

	if c.Uom == nil {
		return 0, nil, apperror.BadRequest("child requires uom when creating new item: " + *c.UniqCode)
	}

	item := &models.Item{
		UniqCode:   *c.UniqCode,
		PartName:   *c.PartName,
		PartNumber: c.PartNumber,
		Model:      c.Model,
		Uom:        *c.Uom,
		Status:     "Draft",
	}
	if err := s.repo.CreateItem(ctx, item); err != nil {
		return 0, nil, err
	}

	revStr := "v1"
	if c.Revision != nil {
		revStr = *c.Revision
	}
	item.CurrentRevision = &revStr
	_ = s.repo.UpdateItem(ctx, item)

	rev := &models.ItemRevision{ItemID: item.ID, Revision: revStr, Status: "Draft"}
	if err := s.repo.CreateRevision(ctx, rev); err != nil {
		return 0, nil, err
	}

	if c.PictureURL != nil {
		_ = s.repo.CreateAsset(ctx, &models.ItemAsset{
			ItemID:    item.ID,
			AssetType: "photo",
			FileURL:   *c.PictureURL,
			Status:    "Active",
		})
	}
	if len(c.ProcessRoutes) > 0 {
		_ = s.createRouting(ctx, item.ID, rev.ID, c.ProcessRoutes)
	}
	if c.MaterialSpec != nil {
		_ = s.saveMaterialSpec(ctx, rev.ID, c.MaterialSpec)
	}

	return item.ID, &rev.ID, nil
}

func (s *service) createRouting(ctx context.Context, itemID, revID int64, routes []models.ProcessRouteInput) error {
	// Validate that submitted routes follow ascending process master sequence.
	processIDs := make([]int64, 0, len(routes))
	seen := make(map[int64]struct{}, len(routes))
	for _, pr := range routes {
		if _, dup := seen[pr.ProcessID]; dup {
			continue
		}
		seen[pr.ProcessID] = struct{}{}
		processIDs = append(processIDs, pr.ProcessID)
	}
	seqMap, err := s.repo.GetProcessSequencesByIDs(ctx, processIDs)
	if err != nil {
		return err
	}
	prevSeq := -1
	for _, pr := range routes {
		seq, ok := seqMap[pr.ProcessID]
		if !ok {
			return apperror.BadRequest(fmt.Sprintf("process_id %d tidak ditemukan (op_seq %d)", pr.ProcessID, pr.OpSeq))
		}
		if seq < prevSeq {
			// return apperror.BadRequest(
			// 	fmt.Sprintf(
			// 		"Urutan proses tidak valid. Routing pada indeks %d (process_id %d) memiliki sequence %d, sedangkan proses sebelumnya memiliki sequence %d. Pastikan urutan routing mengikuti sequence proses dari yang terkecil ke yang terbesar.",
			// 		i, pr.ProcessID, seq, prevSeq,
			// 	),
			// )
			log.Printf(
				"Skip process_id=%d, seq=%d < prevSeq=%d",
				pr.ProcessID, seq, prevSeq,
			)
			continue
		}
		prevSeq = seq
	}

	nextHeaderVersion := 1
	latestHeaders, err := s.repo.GetLatestRoutingHeadersByItemIDs(ctx, []int64{itemID})
	if err != nil {
		return err
	}
	if len(latestHeaders) > 0 && latestHeaders[0].Version >= nextHeaderVersion {
		nextHeaderVersion = latestHeaders[0].Version + 1
	}
	rh := &models.RoutingHeader{ItemID: itemID, ItemRevisionID: &revID, Version: nextHeaderVersion, Status: "Draft"}
	if err := s.repo.CreateRoutingHeader(ctx, rh); err != nil {
		return err
	}
	for _, pr := range routes {
		op := &models.RoutingOperation{
			RoutingHeaderID: rh.ID,
			OpSeq:           pr.OpSeq,
			ProcessID:       pr.ProcessID,
			CycleTimeSec:    pr.CycleTimeSec,
			SetupTimeMin:    pr.SetupTimeMin,
			MachineStroke:   pr.MachineStroke, // free text e.g. "200 spm"
			Notes:           pr.ToolingRef,    // lightweight UI input (dropdown + free text)
			Remark:          pr.Remark,
		}
		if pr.MachineID != nil {
			op.MachineID = pr.MachineID
		}
		if err := s.repo.CreateOperation(ctx, op); err != nil {
			return err
		}
		for _, ti := range pr.Toolings {
			_ = s.repo.CreateTooling(ctx, &models.RoutingOperationTooling{
				RoutingOperationID: op.ID,
				ToolingType:        ti.ToolingType,
				ToolingCode:        ti.ToolingCode,
				ToolingName:        ti.ToolingName,
			})
		}
	}
	return nil
}

func (s *service) saveMaterialSpec(ctx context.Context, revID int64, ms *models.MaterialSpecInput) error {
	spec := &models.ItemMaterialSpec{
		ItemRevisionID: revID,
		MaterialGrade:  ms.MaterialGrade,
		Grade:          ms.Grade,
		TypeMaterial:   ms.TypeMaterial,
		Form:           ms.Form,
		WidthMm:        ms.WidthMm,
		DiameterMm:     ms.DiameterMm,
		ThicknessMm:    ms.ThicknessMm,
		LengthMm:       ms.LengthMm,
		WeightKg:       ms.WeightKg,
		CycleTimeSec:   ms.CycleTimeSec,
		SetupTimeMin:   ms.SetupTimeMin,
		CustomerCycle:  ms.CustomerCycle,
	}
	if ms.SupplierID != nil {
		parsed, err := uuid.Parse(*ms.SupplierID)
		if err != nil {
			return apperror.BadRequest("invalid supplier_id")
		}
		spec.SupplierID = ms.SupplierID
		// lookup name hanya jika supplier_name tidak disertakan di request
		if ms.SupplierName == nil || *ms.SupplierName == "" {
			name := s.repo.GetSupplierName(ctx, parsed)
			if name != "" {
				spec.SupplierName = &name
			}
		}
	}
	if ms.SupplierName != nil && *ms.SupplierName != "" {
		spec.SupplierName = ms.SupplierName
	}
	return s.repo.UpsertMaterialSpec(ctx, spec)
}

// ---------------------------------------------------------------------------
// Detail
// ---------------------------------------------------------------------------

func (s *service) GetBomDetail(ctx context.Context, bomID int64) (*models.BomDetailResponse, error) {
	return s.getBomDetail(ctx, bomID, nil)
}

func (s *service) GetBomDetailByVersion(ctx context.Context, bomID int64, version int) (*models.BomDetailResponse, error) {
	return s.getBomDetail(ctx, bomID, &version)
}

func (s *service) getBomDetail(ctx context.Context, bomID int64, version *int) (*models.BomDetailResponse, error) {
	bom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
		return nil, err
	}
	if version != nil {
		bom, err = s.repo.GetBomByItemAndVersion(ctx, bom.ItemID, *version)
		if err != nil {
			return nil, err
		}
	}

	lines, err := s.repo.GetBomLinesByBomIDs(ctx, []int64{bom.ID})
	if err != nil {
		return nil, err
	}

	preload, err := s.preloadBomData(ctx, []models.BomItem{*bom}, lines)
	if err != nil {
		return nil, err
	}

	parent, ok := preload.items[bom.ItemID]
	if !ok {
		return nil, apperror.NotFound("item tidak ditemukan")
	}

	resp := &models.BomDetailResponse{
		BomID:         bom.ID,
		BomVersion:    bom.Version,
		BomStatus:     bom.Status,
		IsCurrent:     bom.IsCurrent,
		ReadOnly:      bom.Status != "Draft",
		ChangeNote:    bom.ChangeNote,
		ID:            parent.ID,
		UniqCode:      parent.UniqCode,
		PartName:      parent.PartName,
		PartNumber:    parent.PartNumber,
		Model:         parent.Model,
		Status:        parent.Status,
		Description:   bom.Description,
		Asset:         s.buildAssetInfo(preload.assetByItemID(parent.ID)),
		ProcessRoutes: []models.ProcessRouteDetail{},
	}
	if parentRev, ok := preload.revisionForParent(*bom); ok {
		resp.Version = &parentRev.Revision
		if spec, ok := preload.specs[parentRev.ID]; ok {
			resp.MaterialSpec = s.toSpecDetail(&spec)
		}
		if routes, ok := preload.routesByRevID[parentRev.ID]; ok {
			resp.ProcessRoutes = routes
		}
	}

	resp.Children = s.buildDetailTree(lines, preload, parent.ID, 1)

	return resp, nil
}

func (s *service) GetBomVersions(ctx context.Context, bomID int64) (*models.BomVersionsResponse, error) {
	bom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
		return nil, err
	}
	rootItem, err := s.repo.GetItemByID(ctx, bom.ItemID)
	if err != nil {
		return nil, err
	}
	versions, err := s.repo.GetBomVersionsByItemID(ctx, bom.ItemID)
	if err != nil {
		return nil, err
	}
	resp := &models.BomVersionsResponse{
		RootItemID:   rootItem.ID,
		RootItemCode: rootItem.UniqCode,
		RootItemName: rootItem.PartName,
		Versions:     make([]models.BomVersionOption, 0, len(versions)),
	}
	for _, version := range versions {
		option := models.BomVersionOption{
			BomID:      version.ID,
			BomVersion: version.Version,
			Label:      fmt.Sprintf("v%d", version.Version),
			BomStatus:  version.Status,
			IsCurrent:  version.IsCurrent,
			ReadOnly:   version.Status != "Draft",
			ChangeNote: version.ChangeNote,
			CreatedAt:  version.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
		}
		if version.IsCurrent {
			resp.CurrentBomID = &version.ID
			resp.CurrentVersion = &version.Version
		}
		resp.Versions = append(resp.Versions, option)
	}
	return resp, nil
}

func (s *service) ensureDraftBom(bom *models.BomItem) error {
	if bom.Status != "Draft" {
		return apperror.Conflict("version is read-only")
	}
	return nil
}

func (s *service) resolveBomRootRevision(ctx context.Context, bom *models.BomItem) (*models.ItemRevision, error) {
	if bom.RootItemRevisionID != nil {
		rev, err := s.repo.GetRevisionByID(ctx, *bom.RootItemRevisionID)
		if err == nil {
			return rev, nil
		}
	}
	return s.repo.GetLatestRevision(ctx, bom.ItemID)
}

func (s *service) CreateBomRevision(ctx context.Context, bomID int64, req models.CreateBomRevisionRequest) (*models.CreateBomRevisionResponse, error) {
	sourceBom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
		return nil, err
	}
	if req.SourceVersion != nil {
		sourceBom, err = s.repo.GetBomByItemAndVersion(ctx, sourceBom.ItemID, *req.SourceVersion)
		if err != nil {
			return nil, err
		}
	}
	versions, err := s.repo.GetBomVersionsByItemID(ctx, sourceBom.ItemID)
	if err != nil {
		return nil, err
	}
	nextVersion := sourceBom.Version + 1
	for _, version := range versions {
		if version.Version >= nextVersion {
			nextVersion = version.Version + 1
		}
		if version.Status == "Draft" {
			return nil, apperror.Conflict("another draft already exists")
		}
	}
	sourceRev, err := s.resolveBomRootRevision(ctx, sourceBom)
	if err != nil {
		return nil, err
	}
	if sourceRev == nil {
		return nil, apperror.NotFound("source item revision tidak ditemukan")
	}
	newRev, err := s.createNextItemRevision(ctx, sourceBom.ItemID, nextVersion, req.ChangeNote)
	if err != nil {
		return nil, err
	}
	newRevLabel := newRev.Revision
	rootItem, err := s.repo.GetItemByID(ctx, sourceBom.ItemID)
	if err != nil {
		return nil, err
	}
	rootItem.CurrentRevision = &newRevLabel
	if err := s.repo.UpdateItem(ctx, rootItem); err != nil {
		return nil, err
	}
	if spec, err := s.repo.GetMaterialSpec(ctx, sourceRev.ID); err != nil {
		return nil, err
	} else if spec != nil {
		copySpec := &models.ItemMaterialSpec{
			ItemRevisionID: newRev.ID,
			MaterialGrade:  spec.MaterialGrade,
			Form:           spec.Form,
			WidthMm:        spec.WidthMm,
			DiameterMm:     spec.DiameterMm,
			ThicknessMm:    spec.ThicknessMm,
			LengthMm:       spec.LengthMm,
			WeightKg:       spec.WeightKg,
			SupplierID:     spec.SupplierID,
			SupplierName:   spec.SupplierName,
			CycleTimeSec:   spec.CycleTimeSec,
			SetupTimeMin:   spec.SetupTimeMin,
			CustomerCycle:  spec.CustomerCycle,
		}
		if err := s.repo.UpsertMaterialSpec(ctx, copySpec); err != nil {
			return nil, err
		}
	}
	if err := s.cloneRoutingForRevision(ctx, sourceBom.ItemID, sourceRev.ID, newRev.ID); err != nil {
		return nil, err
	}
	for i := range versions {
		if versions[i].IsCurrent {
			versions[i].IsCurrent = false
			if err := s.repo.UpdateBomItem(ctx, &versions[i]); err != nil {
				return nil, err
			}
		}
	}
	newBom := &models.BomItem{
		ItemID:             sourceBom.ItemID,
		RootItemRevisionID: &newRev.ID,
		CopiedFromBomID:    &sourceBom.ID,
		Version:            nextVersion,
		Status:             "Released",
		Description:        sourceBom.Description,
		ChangeNote:         req.ChangeNote,
		IsCurrent:          true,
	}
	if err := s.repo.CreateBomItem(ctx, newBom); err != nil {
		return nil, err
	}
	lines, err := s.repo.GetBomLines(ctx, sourceBom.ID)
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		clone := line
		clone.ID = 0
		clone.BomItemID = newBom.ID
		if err := s.repo.CreateBomLine(ctx, &clone); err != nil {
			return nil, err
		}
	}
	return &models.CreateBomRevisionResponse{
		SourceBomID:   sourceBom.ID,
		SourceVersion: sourceBom.Version,
		BomID:         newBom.ID,
		BomVersion:    newBom.Version,
		BomStatus:     newBom.Status,
		IsCurrent:     newBom.IsCurrent,
		ReadOnly:      newBom.Status != "Draft",
		ChangeNote:    newBom.ChangeNote,
		Message:       fmt.Sprintf("BOM revision created from v%d", sourceBom.Version),
	}, nil
}

func (s *service) createNextItemRevision(ctx context.Context, itemID int64, baseVersion int, changeNote *string) (*models.ItemRevision, error) {
	startVersion := baseVersion
	latestRev, err := s.repo.GetLatestRevision(ctx, itemID)
	if err != nil {
		return nil, err
	}
	if latestRev != nil {
		if parsed, ok := parseRevisionNumber(latestRev.Revision); ok && parsed >= startVersion {
			startVersion = parsed + 1
		}
	}
	for version := startVersion; version < startVersion+20; version++ {
		rev := &models.ItemRevision{
			ItemID:     itemID,
			Revision:   fmt.Sprintf("v%d", version),
			Status:     "Draft",
			ChangeNote: changeNote,
		}
		if err := s.repo.CreateRevision(ctx, rev); err != nil {
			if appErr, ok := apperror.As(err); ok && appErr.Code == apperror.CodeConflict {
				continue
			}
			return nil, err
		}
		return rev, nil
	}
	return nil, apperror.Conflict("unable to allocate next revision label")
}

func parseRevisionNumber(revision string) (int, bool) {
	revision = strings.TrimSpace(strings.ToLower(revision))
	revision = strings.TrimPrefix(revision, "v")
	if revision == "" {
		return 0, false
	}
	value, err := strconv.Atoi(revision)
	if err != nil {
		return 0, false
	}
	return value, true
}

func (s *service) cloneRoutingForRevision(ctx context.Context, itemID, sourceRevisionID, targetRevisionID int64) error {
	header, err := s.repo.GetRoutingHeaderByRevisionID(ctx, sourceRevisionID)
	if err != nil {
		return err
	}
	if header == nil {
		return nil
	}
	ops, err := s.repo.GetRoutingOperationsByHeaderIDs(ctx, []int64{header.ID})
	if err != nil {
		return err
	}
	opIDs := make([]int64, 0, len(ops))
	for _, op := range ops {
		opIDs = append(opIDs, op.ID)
	}
	toolings, err := s.repo.GetToolingsByOperationIDs(ctx, opIDs)
	if err != nil {
		return err
	}
	nextHeaderVersion := header.Version + 1
	latestHeaders, err := s.repo.GetLatestRoutingHeadersByItemIDs(ctx, []int64{itemID})
	if err != nil {
		return err
	}
	if len(latestHeaders) > 0 && latestHeaders[0].Version >= nextHeaderVersion {
		nextHeaderVersion = latestHeaders[0].Version + 1
	}
	newHeader := &models.RoutingHeader{ItemID: itemID, ItemRevisionID: &targetRevisionID, Version: nextHeaderVersion, Status: "Draft"}
	if err := s.repo.CreateRoutingHeader(ctx, newHeader); err != nil {
		return err
	}
	toolingsByOpID := make(map[int64][]models.RoutingOperationTooling)
	for _, tooling := range toolings {
		toolingsByOpID[tooling.RoutingOperationID] = append(toolingsByOpID[tooling.RoutingOperationID], tooling)
	}
	for _, op := range ops {
		newOp := &models.RoutingOperation{
			RoutingHeaderID: newHeader.ID,
			OpSeq:           op.OpSeq,
			ProcessID:       op.ProcessID,
			MachineID:       op.MachineID,
			CycleTimeSec:    op.CycleTimeSec,
			SetupTimeMin:    op.SetupTimeMin,
			MachineStroke:   op.MachineStroke,
			Notes:           op.Notes,
		}
		if err := s.repo.CreateOperation(ctx, newOp); err != nil {
			return err
		}
		for _, tooling := range toolingsByOpID[op.ID] {
			if err := s.repo.CreateTooling(ctx, &models.RoutingOperationTooling{
				RoutingOperationID: newOp.ID,
				ToolingType:        tooling.ToolingType,
				ToolingCode:        tooling.ToolingCode,
				ToolingName:        tooling.ToolingName,
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *service) ActivateBomVersion(ctx context.Context, bomID int64) (*models.BomDetailResponse, error) {
	bom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
		return nil, err
	}
	versions, err := s.repo.GetBomVersionsByItemID(ctx, bom.ItemID)
	if err != nil {
		return nil, err
	}
	for i := range versions {
		if versions[i].ID == bomID {
			continue
		}
		if versions[i].IsCurrent {
			versions[i].IsCurrent = false
			if err := s.repo.UpdateBomItem(ctx, &versions[i]); err != nil {
				return nil, err
			}
		}
	}
	bom.IsCurrent = true
	if err := s.repo.UpdateBomItem(ctx, bom); err != nil {
		return nil, err
	}
	return s.GetBomDetail(ctx, bomID)
}

func (s *service) AddProcessRoute(ctx context.Context, bomID int64, reqs []models.AddProcessRouteRequest) ([]models.ProcessRouteMutationResponse, error) {
	bom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
		return nil, err
	}
	versions, err := s.repo.GetBomVersionsByItemID(ctx, bom.ItemID)
	if err != nil {
		return nil, err
	}

	// Find the latest version (highest bom_version number)
	var latest *models.BomItem
	for i := range versions {
		if latest == nil || versions[i].Version > latest.Version {
			latest = &versions[i]
		}
	}
	if latest == nil {
		return nil, apperror.NotFound("no bom versions found")
	}

	// Only the latest version is editable; older versions are read-only
	if bomID != latest.ID {
		return nil, apperror.Conflict("version is read-only")
	}

	// If latest is Released, create a new Draft revision from it
	workingBom := latest
	if latest.Status != "Draft" {
		workingBom, err = s.createDraftRevisionFrom(ctx, latest, versions)
		if err != nil {
			return nil, err
		}
	}

	// Add routes to the working bom
	results := make([]models.ProcessRouteMutationResponse, 0, len(reqs))
	for _, req := range reqs {
		targetItemID, targetRevisionID, lineID, err := s.resolveRouteTarget(ctx, workingBom, req.LineID)
		if err != nil {
			return nil, err
		}
		header, err := s.repo.GetRoutingHeaderByRevisionID(ctx, targetRevisionID)
		if err != nil {
			return nil, err
		}
		if header == nil {
			header = &models.RoutingHeader{ItemID: targetItemID, ItemRevisionID: &targetRevisionID, Version: 1, Status: "Draft"}
			if err := s.repo.CreateRoutingHeader(ctx, header); err != nil {
				return nil, err
			}
		}
		op := &models.RoutingOperation{
			RoutingHeaderID: header.ID,
			OpSeq:           req.OpSeq,
			ProcessID:       req.ProcessID,
			MachineID:       req.MachineID,
			CycleTimeSec:    req.CycleTimeSec,
			SetupTimeMin:    req.SetupTimeMin,
			MachineStroke:   req.MachineStroke,
			Notes:           req.ToolingRef,
		}
		if err := s.repo.CreateOperation(ctx, op); err != nil {
			return nil, err
		}
		if err := s.repo.ReplaceToolings(ctx, op.ID, req.Toolings); err != nil {
			return nil, err
		}
		res, err := s.buildProcessRouteMutationResponse(ctx, workingBom, lineID, op)
		if err != nil {
			return nil, err
		}
		results = append(results, *res)
	}

	// Auto-release: unset IsCurrent on all versions, set workingBom as Released + IsCurrent
	for i := range versions {
		if versions[i].ID == workingBom.ID {
			continue
		}
		if versions[i].IsCurrent {
			versions[i].IsCurrent = false
			if err := s.repo.UpdateBomItem(ctx, &versions[i]); err != nil {
				return nil, err
			}
		}
	}
	workingBom.Status = "Released"
	workingBom.IsCurrent = true
	if err := s.repo.UpdateBomItem(ctx, workingBom); err != nil {
		return nil, err
	}

	return results, nil
}

// createDraftRevisionFrom creates a new Draft BomItem by cloning sourceBom (routing + spec + lines).
func (s *service) createDraftRevisionFrom(ctx context.Context, sourceBom *models.BomItem, versions []models.BomItem) (*models.BomItem, error) {
	nextVersion := sourceBom.Version + 1
	for _, v := range versions {
		if v.Version >= nextVersion {
			nextVersion = v.Version + 1
		}
	}
	sourceRev, err := s.resolveBomRootRevision(ctx, sourceBom)
	if err != nil {
		return nil, err
	}
	if sourceRev == nil {
		return nil, apperror.NotFound("source item revision tidak ditemukan")
	}
	newRev, err := s.createNextItemRevision(ctx, sourceBom.ItemID, nextVersion, nil)
	if err != nil {
		return nil, err
	}
	rootItem, err := s.repo.GetItemByID(ctx, sourceBom.ItemID)
	if err != nil {
		return nil, err
	}
	newRevLabel := newRev.Revision
	rootItem.CurrentRevision = &newRevLabel
	_ = s.repo.UpdateItem(ctx, rootItem)

	if spec, err := s.repo.GetMaterialSpec(ctx, sourceRev.ID); err == nil && spec != nil {
		copySpec := *spec
		copySpec.ItemRevisionID = newRev.ID
		_ = s.repo.UpsertMaterialSpec(ctx, &copySpec)
	}
	if err := s.cloneRoutingForRevision(ctx, sourceBom.ItemID, sourceRev.ID, newRev.ID); err != nil {
		return nil, err
	}
	newBom := &models.BomItem{
		ItemID:             sourceBom.ItemID,
		RootItemRevisionID: &newRev.ID,
		CopiedFromBomID:    &sourceBom.ID,
		Version:            nextVersion,
		Status:             "Draft",
		Description:        sourceBom.Description,
		IsCurrent:          false,
	}
	if err := s.repo.CreateBomItem(ctx, newBom); err != nil {
		return nil, err
	}
	lines, err := s.repo.GetBomLines(ctx, sourceBom.ID)
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		clone := line
		clone.ID = 0
		clone.BomItemID = newBom.ID
		_ = s.repo.CreateBomLine(ctx, &clone)
	}
	return newBom, nil
}

func (s *service) PatchProcessRoute(ctx context.Context, bomID, routeID int64, req models.PatchProcessRouteRequest) (*models.ProcessRouteMutationResponse, error) {
	bom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureDraftBom(bom); err != nil {
		return nil, err
	}
	op, err := s.repo.GetOperationByID(ctx, routeID)
	if err != nil {
		return nil, err
	}
	allowedHeaderIDs, err := s.allowedRouteHeaderIDs(ctx, bom)
	if err != nil {
		return nil, err
	}
	if _, ok := allowedHeaderIDs[op.RoutingHeaderID]; !ok {
		return nil, apperror.NotFound("routing operation tidak ditemukan in bom version")
	}
	if req.OpSeq != nil {
		op.OpSeq = *req.OpSeq
	}
	if req.ProcessID != nil {
		op.ProcessID = *req.ProcessID
	}
	if req.MachineID != nil {
		op.MachineID = req.MachineID
	}
	if req.CycleTimeSec != nil {
		op.CycleTimeSec = req.CycleTimeSec
	}
	if req.SetupTimeMin != nil {
		op.SetupTimeMin = req.SetupTimeMin
	}
	if req.MachineStroke != nil {
		op.MachineStroke = req.MachineStroke
	}
	if req.ToolingRef != nil {
		op.Notes = req.ToolingRef
	}
	if err := s.repo.UpdateOperation(ctx, op); err != nil {
		return nil, err
	}
	if req.Toolings != nil {
		if err := s.repo.ReplaceToolings(ctx, op.ID, *req.Toolings); err != nil {
			return nil, err
		}
	}
	return s.buildProcessRouteMutationResponse(ctx, bom, nil, op)
}

func (s *service) allowedRouteHeaderIDs(ctx context.Context, bom *models.BomItem) (map[int64]struct{}, error) {
	allowed := make(map[int64]struct{})
	if bom.RootItemRevisionID != nil {
		header, err := s.repo.GetRoutingHeaderByRevisionID(ctx, *bom.RootItemRevisionID)
		if err != nil {
			return nil, err
		}
		if header != nil {
			allowed[header.ID] = struct{}{}
		}
	}
	lines, err := s.repo.GetBomLines(ctx, bom.ID)
	if err != nil {
		return nil, err
	}
	for _, line := range lines {
		if line.ChildItemRevisionID == nil {
			continue
		}
		header, err := s.repo.GetRoutingHeaderByRevisionID(ctx, *line.ChildItemRevisionID)
		if err != nil {
			return nil, err
		}
		if header != nil {
			allowed[header.ID] = struct{}{}
		}
	}
	return allowed, nil
}

func (s *service) resolveRouteTarget(ctx context.Context, bom *models.BomItem, lineID *int64) (int64, int64, *int64, error) {
	if lineID == nil {
		rev, err := s.resolveBomRootRevision(ctx, bom)
		if err != nil {
			return 0, 0, nil, err
		}
		if rev == nil {
			return 0, 0, nil, apperror.NotFound("root item revision tidak ditemukan")
		}
		return bom.ItemID, rev.ID, nil, nil
	}
	line, err := s.repo.GetBomLineByID(ctx, bom.ID, *lineID)
	if err != nil {
		return 0, 0, nil, err
	}
	if line.ChildItemRevisionID != nil {
		return line.ChildItemID, *line.ChildItemRevisionID, lineID, nil
	}
	rev, err := s.repo.GetLatestRevision(ctx, line.ChildItemID)
	if err != nil {
		return 0, 0, nil, err
	}
	if rev == nil {
		return 0, 0, nil, apperror.NotFound("child item revision tidak ditemukan")
	}
	return line.ChildItemID, rev.ID, lineID, nil
}

func (s *service) buildProcessRouteMutationResponse(ctx context.Context, bom *models.BomItem, lineID *int64, op *models.RoutingOperation) (*models.ProcessRouteMutationResponse, error) {
	processName := s.repo.GetProcessName(ctx, op.ProcessID)
	var machineName *string
	if op.MachineID != nil {
		name := s.repo.GetMachineName(ctx, *op.MachineID)
		machineName = &name
	}
	return &models.ProcessRouteMutationResponse{
		RouteID:       op.ID,
		BomID:         bom.ID,
		BomVersion:    bom.Version,
		LineID:        lineID,
		OpSeq:         op.OpSeq,
		ProcessID:     op.ProcessID,
		ProcessName:   processName,
		MachineID:     op.MachineID,
		MachineName:   machineName,
		CycleTimeSec:  op.CycleTimeSec,
		SetupTimeMin:  op.SetupTimeMin,
		MachineStroke: op.MachineStroke,
		ToolingRef:    op.Notes,
	}, nil
}

func (s *service) UpdateBom(ctx context.Context, bomID int64, req models.UpdateBomRequest) (*models.BomDetailResponse, error) {
	bom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureDraftBom(bom); err != nil {
		return nil, err
	}
	item, err := s.repo.GetItemByID(ctx, bom.ItemID)
	if err != nil {
		return nil, err
	}

	// Update item fields
	itemChanged := false
	if req.PartName != nil {
		item.PartName = *req.PartName
		itemChanged = true
	}
	if req.PartNumber != nil {
		item.PartNumber = req.PartNumber
		itemChanged = true
	}
	if req.Model != nil {
		item.Model = req.Model
		itemChanged = true
	}
	if req.Status != nil {
		item.Status = *req.Status
		itemChanged = true
	}
	if itemChanged {
		if err := s.repo.UpdateItem(ctx, item); err != nil {
			return nil, err
		}
	}

	// Update BOM header fields
	bomChanged := false
	if req.Description != nil {
		bom.Description = req.Description
		bomChanged = true
	}
	if req.BomStatus != nil {
		bom.Status = *req.BomStatus
		bomChanged = true
	}
	if bomChanged {
		if err := s.repo.UpdateBomItem(ctx, bom); err != nil {
			return nil, err
		}
	}

	// Replace picture
	if req.PictureURL != nil {
		if err := s.repo.UpsertItemAssetURL(ctx, item.ID, "photo", *req.PictureURL); err != nil {
			return nil, err
		}
	}

	// Replace process routes when key is explicitly provided
	if req.ProcessRoutes != nil {
		rev, err := s.resolveBomRootRevision(ctx, bom)
		if err != nil {
			return nil, err
		}
		if rev != nil {
			if err := s.repo.DeleteRoutingByRevisionID(ctx, rev.ID); err != nil {
				return nil, err
			}
			if len(*req.ProcessRoutes) > 0 {
				if err := s.createRouting(ctx, item.ID, rev.ID, *req.ProcessRoutes); err != nil {
					return nil, err
				}
			}
		}
	}

	// Upsert material spec when provided
	if req.MaterialSpec != nil {
		rev, err := s.resolveBomRootRevision(ctx, bom)
		if err != nil {
			return nil, err
		}
		if rev != nil {
			if err := s.saveMaterialSpec(ctx, rev.ID, req.MaterialSpec); err != nil {
				return nil, err
			}
		}
	}

	return s.GetBomDetail(ctx, bomID)
}

func (s *service) UpdateBomChild(ctx context.Context, bomID, lineID int64, req models.UpdateBomChildRequest) (*models.BomDetailResponse, error) {
	bom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
		return nil, err
	}
	if err := s.ensureDraftBom(bom); err != nil {
		return nil, err
	}

	// Load the target line (validates it belongs to this BOM)
	line, err := s.repo.GetBomLineByID(ctx, bomID, lineID)
	if err != nil {
		return nil, err
	}

	// Update BomLine fields
	lineChanged := false
	if req.QtyPerUniq != nil {
		line.QtyPerUniq = *req.QtyPerUniq
		lineChanged = true
	}
	if req.ScrapFactor != nil {
		line.ScrapFactor = *req.ScrapFactor
		lineChanged = true
	}
	if req.IsPhantom != nil {
		line.IsPhantom = *req.IsPhantom
		lineChanged = true
	}
	if lineChanged {
		if err := s.repo.UpdateBomLine(ctx, line); err != nil {
			return nil, err
		}
	}

	// Load child item
	item, err := s.repo.GetItemByID(ctx, line.ChildItemID)
	if err != nil {
		return nil, err
	}

	// Update child Item fields
	itemChanged := false
	if req.PartName != nil {
		item.PartName = *req.PartName
		itemChanged = true
	}
	if req.PartNumber != nil {
		item.PartNumber = req.PartNumber
		itemChanged = true
	}
	if req.Status != nil {
		item.Status = *req.Status
		itemChanged = true
	}
	if itemChanged {
		if err := s.repo.UpdateItem(ctx, item); err != nil {
			return nil, err
		}
	}

	// Replace picture
	if req.PictureURL != nil {
		if err := s.repo.UpsertItemAssetURL(ctx, item.ID, "photo", *req.PictureURL); err != nil {
			return nil, err
		}
	}

	// Replace process routes when key is explicitly provided
	if req.ProcessRoutes != nil {
		var revID int64
		if line.ChildItemRevisionID != nil {
			revID = *line.ChildItemRevisionID
		} else {
			rev, err := s.repo.GetLatestRevision(ctx, item.ID)
			if err != nil {
				return nil, err
			}
			if rev != nil {
				revID = rev.ID
			}
		}
		if revID != 0 {
			if err := s.repo.DeleteRoutingByRevisionID(ctx, revID); err != nil {
				return nil, err
			}
			if len(*req.ProcessRoutes) > 0 {
				if err := s.createRouting(ctx, item.ID, revID, *req.ProcessRoutes); err != nil {
					return nil, err
				}
			}
		}
	}

	// Upsert material spec when provided
	if req.MaterialSpec != nil {
		if line.ChildItemRevisionID != nil {
			if err := s.saveMaterialSpec(ctx, *line.ChildItemRevisionID, req.MaterialSpec); err != nil {
				return nil, err
			}
		} else {
			rev, err := s.repo.GetLatestRevision(ctx, item.ID)
			if err != nil {
				return nil, err
			}
			if rev != nil {
				if err := s.saveMaterialSpec(ctx, rev.ID, req.MaterialSpec); err != nil {
					return nil, err
				}
			}
		}
	}

	return s.GetBomDetail(ctx, bomID)
}

func (s *service) DeleteBom(ctx context.Context, bomID int64) error {
	// [items-soft-delete]
	// Sebelumnya DeleteBom hanya menghapus baris `bom_item`, sehingga
	// baris `items` yang bersangkutan tetap ada dengan `deleted_at = NULL`.
	// Akibatnya user tidak bisa membuat item baru dengan `uniq_code`
	// yang sama karena constraint UNIQUE `items.uniq_code`.
	//
	// Sekarang:
	//   1. Ambil bom_item untuk tahu `item_id` parent.
	//   2. Hapus bom_item (hard delete — history tetap terjaga di
	//      approval_audits/logs kalau ada).
	//   3. Soft-delete item parent (isi `deleted_at = NOW()`).
	// Dengan migration 0096 yang mengubah UNIQUE constraint jadi partial
	// unique index `WHERE deleted_at IS NULL`, uniq_code otomatis bebas
	// dipakai ulang setelah item di-soft-delete.
	bom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
		return err
	}
	if err := s.repo.DeleteBomItem(ctx, bomID); err != nil {
		return err
	}
	if err := s.repo.SoftDeleteItem(ctx, bom.ItemID); err != nil {
		return err
	}
	return nil
}

func (s *service) DeleteBomChild(ctx context.Context, bomID, childItemID int64) (int64, error) {
	if _, err := s.repo.GetBomByID(ctx, bomID); err != nil {
		return 0, err
	}

	lines, err := s.repo.GetBomLines(ctx, bomID)
	if err != nil {
		return 0, err
	}

	roots := make([]models.BomLine, 0)
	childrenByParentLevel := make(map[lineTreeKey][]models.BomLine)
	for _, line := range lines {
		childrenByParentLevel[lineTreeKey{parentItemID: line.ParentItemID, level: line.Level}] = append(childrenByParentLevel[lineTreeKey{parentItemID: line.ParentItemID, level: line.Level}], line)
		if line.ChildItemID == childItemID {
			roots = append(roots, line)
		}
	}

	if len(roots) == 0 {
		return 0, apperror.NotFound("child item tidak ditemukan in bom")
	}

	lineIDs := collectSubtreeLineIDs(lines, roots)
	deleted, err := s.repo.DeleteBomLinesByIDs(ctx, bomID, lineIDs)
	if err != nil {
		return 0, err
	}
	if deleted == 0 {
		return 0, apperror.NotFound("child item tidak ditemukan in bom")
	}

	return deleted, nil
}

func (s *service) DeleteBomLine(ctx context.Context, bomID, lineID int64) (int64, error) {
	if _, err := s.repo.GetBomByID(ctx, bomID); err != nil {
		return 0, err
	}

	lines, err := s.repo.GetBomLines(ctx, bomID)
	if err != nil {
		return 0, err
	}

	var root *models.BomLine
	for i := range lines {
		if lines[i].ID == lineID {
			root = &lines[i]
			break
		}
	}
	if root == nil {
		return 0, apperror.NotFound("line tidak ditemukan in bom")
	}

	lineIDs := collectSubtreeLineIDs(lines, []models.BomLine{*root})
	deleted, err := s.repo.DeleteBomLinesByIDs(ctx, bomID, lineIDs)
	if err != nil {
		return 0, err
	}
	if deleted == 0 {
		return 0, apperror.NotFound("line tidak ditemukan in bom")
	}

	return deleted, nil
}

func collectSubtreeLineIDs(lines []models.BomLine, roots []models.BomLine) []int64 {
	childrenByParentLevel := make(map[lineTreeKey][]models.BomLine)
	for _, line := range lines {
		childrenByParentLevel[lineTreeKey{parentItemID: line.ParentItemID, level: line.Level}] = append(childrenByParentLevel[lineTreeKey{parentItemID: line.ParentItemID, level: line.Level}], line)
	}

	deleteSet := make(map[int64]struct{})
	queue := make([]models.BomLine, 0, len(roots))
	queue = append(queue, roots...)

	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]

		if _, seen := deleteSet[curr.ID]; seen {
			continue
		}
		deleteSet[curr.ID] = struct{}{}

		next := childrenByParentLevel[lineTreeKey{parentItemID: curr.ChildItemID, level: curr.Level + 1}]
		if len(next) > 0 {
			queue = append(queue, next...)
		}
	}

	lineIDs := make([]int64, 0, len(deleteSet))
	for lineID := range deleteSet {
		lineIDs = append(lineIDs, lineID)
	}
	sort.Slice(lineIDs, func(i, j int) bool { return lineIDs[i] < lineIDs[j] })
	return lineIDs
}

func (s *service) buildDetailTree(lines []models.BomLine, preload *bomPreload, parentItemID int64, level int16) []models.BomDetailChild {
	children := preload.childrenByParent(parentItemID, level, lines)
	rows := make([]models.BomDetailChild, 0, len(children))
	for _, line := range children {
		child, ok := preload.items[line.ChildItemID]
		if !ok {
			continue
		}

		row := models.BomDetailChild{
			ID:            child.ID,
			LineID:        line.ID,
			UniqCode:      child.UniqCode,
			PartName:      child.PartName,
			PartNumber:    child.PartNumber,
			Model:         child.Model,
			Level:         level,
			QPU:           line.QtyPerUniq,
			Asset:         s.buildAssetInfo(preload.assetByItemID(child.ID)),
			Status:        child.Status,
			ProcessRoutes: []models.ProcessRouteDetail{},
		}
		if rev, ok := preload.revisionForChild(line, child.ID); ok {
			row.Version = &rev.Revision
			if spec, ok := preload.specs[rev.ID]; ok {
				row.MaterialSpec = s.toSpecDetail(&spec)
			}
			if routes, ok := preload.routesByRevID[rev.ID]; ok {
				row.ProcessRoutes = routes
			}
		}
		if level < 6 {
			row.Children = s.buildDetailTree(lines, preload, child.ID, level+1)
		}
		rows = append(rows, row)
	}
	return rows
}

func (s *service) toSpecDetail(spec *models.ItemMaterialSpec) *models.MaterialSpecDetail {
	d := &models.MaterialSpecDetail{
		MaterialGrade: spec.MaterialGrade,
		Grade:         spec.Grade,
		TypeMaterial:  spec.TypeMaterial,
		Form:          spec.Form,
		WidthMm:       spec.WidthMm,
		DiameterMm:    spec.DiameterMm,
		ThicknessMm:   spec.ThicknessMm,
		LengthMm:      spec.LengthMm,
		WeightKg:      spec.WeightKg,
		CycleTimeSec:  spec.CycleTimeSec,
		SetupTimeMin:  spec.SetupTimeMin,
		CustomerCycle: spec.CustomerCycle,
		SupplierName:  spec.SupplierName,
	}
	return d
}

func (s *service) toRouteDetails(
	ops []models.RoutingOperation,
	toolings []models.RoutingOperationTooling,
	processNames map[int64]string,
	machineNames map[int64]string,
) []models.ProcessRouteDetail {
	// Index toolings by op ID
	tMap := make(map[int64][]models.RoutingOperationTooling)
	for _, t := range toolings {
		tMap[t.RoutingOperationID] = append(tMap[t.RoutingOperationID], t)
	}

	details := make([]models.ProcessRouteDetail, 0, len(ops))
	for _, op := range ops {
		d := models.ProcessRouteDetail{
			RouteID:       op.ID,
			OpSeq:         op.OpSeq,
			ProcessID:     op.ProcessID,
			ProcessName:   processNames[op.ProcessID],
			MachineID:     op.MachineID,
			CycleTimeSec:  op.CycleTimeSec,
			SetupTimeMin:  op.SetupTimeMin,
			MachineStroke: op.MachineStroke,
			Remark:        op.Remark,
			ToolingRef:    op.Notes,
		}
		if op.MachineID != nil {
			if name, ok := machineNames[*op.MachineID]; ok {
				d.MachineName = &name
			}
		}
		for _, t := range tMap[op.ID] {
			d.Toolings = append(d.Toolings, models.ToolingDetail{
				ToolingType: t.ToolingType,
				ToolingCode: t.ToolingCode,
				ToolingName: t.ToolingName,
			})
		}
		details = append(details, d)
	}
	return details
}

func (s *service) preloadBomData(ctx context.Context, bomItems []models.BomItem, lines []models.BomLine) (*bomPreload, error) {
	itemIDSet := make(map[int64]struct{})
	for _, bom := range bomItems {
		itemIDSet[bom.ItemID] = struct{}{}
	}
	children := make(map[lineTreeKey][]models.BomLine)
	for _, line := range lines {
		itemIDSet[line.ParentItemID] = struct{}{}
		itemIDSet[line.ChildItemID] = struct{}{}
		key := lineTreeKey{parentItemID: line.ParentItemID, level: line.Level}
		children[key] = append(children[key], line)
	}

	itemIDs := uniqueInt64Keys(itemIDSet)
	items, err := s.repo.GetItemsByIDs(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	itemMap := make(map[int64]models.Item, len(items))
	for _, item := range items {
		itemMap[item.ID] = item
	}

	revisions, err := s.repo.GetLatestRevisionsByItemIDs(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	latestRevisionMap := make(map[int64]models.ItemRevision, len(revisions))
	revisionIDSet := make(map[int64]struct{}, len(revisions))
	for _, revision := range revisions {
		latestRevisionMap[revision.ItemID] = revision
		revisionIDSet[revision.ID] = struct{}{}
	}
	for _, bom := range bomItems {
		if bom.RootItemRevisionID != nil {
			revisionIDSet[*bom.RootItemRevisionID] = struct{}{}
		}
	}
	for _, line := range lines {
		if line.ChildItemRevisionID != nil {
			revisionIDSet[*line.ChildItemRevisionID] = struct{}{}
		}
	}

	revisionsByID, err := s.repo.GetRevisionsByIDs(ctx, uniqueInt64Keys(revisionIDSet))
	if err != nil {
		return nil, err
	}
	revisionByID := make(map[int64]models.ItemRevision, len(revisionsByID))
	for _, revision := range revisionsByID {
		revisionByID[revision.ID] = revision
	}

	assets, err := s.repo.GetFirstAssetsByItemIDs(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	assetMap := make(map[int64]models.ItemAsset, len(assets))
	for _, asset := range assets {
		assetMap[asset.ItemID] = asset
	}

	specs, err := s.repo.GetMaterialSpecsByRevisionIDs(ctx, uniqueInt64Keys(revisionIDSet))
	if err != nil {
		return nil, err
	}
	specMap := make(map[int64]models.ItemMaterialSpec, len(specs))
	for _, spec := range specs {
		specMap[spec.ItemRevisionID] = spec
	}

	revisionIDs := make([]int64, 0, len(revisionByID))
	for revisionID := range revisionByID {
		revisionIDs = append(revisionIDs, revisionID)
	}
	headers, err := s.repo.GetRoutingHeadersByRevisionIDs(ctx, revisionIDs)
	if err != nil {
		return nil, err
	}
	headerIDs := make([]int64, 0, len(headers))
	headerRevisionMap := make(map[int64]int64, len(headers))
	for _, header := range headers {
		headerIDs = append(headerIDs, header.ID)
		if header.ItemRevisionID != nil {
			headerRevisionMap[header.ID] = *header.ItemRevisionID
		}
	}

	operations, err := s.repo.GetRoutingOperationsByHeaderIDs(ctx, headerIDs)
	if err != nil {
		return nil, err
	}
	opIDs := make([]int64, 0, len(operations))
	opByHeaderID := make(map[int64][]models.RoutingOperation)
	processIDSet := make(map[int64]struct{})
	machineIDSet := make(map[int64]struct{})
	for _, operation := range operations {
		opIDs = append(opIDs, operation.ID)
		opByHeaderID[operation.RoutingHeaderID] = append(opByHeaderID[operation.RoutingHeaderID], operation)
		processIDSet[operation.ProcessID] = struct{}{}
		if operation.MachineID != nil {
			machineIDSet[*operation.MachineID] = struct{}{}
		}
	}

	toolings, err := s.repo.GetToolingsByOperationIDs(ctx, opIDs)
	if err != nil {
		return nil, err
	}
	toolingsByOpID := make(map[int64][]models.RoutingOperationTooling)
	for _, tooling := range toolings {
		toolingsByOpID[tooling.RoutingOperationID] = append(toolingsByOpID[tooling.RoutingOperationID], tooling)
	}

	processNames, err := s.repo.GetProcessNamesByIDs(ctx, uniqueInt64Keys(processIDSet))
	if err != nil {
		return nil, err
	}
	machineNames, err := s.repo.GetMachineNamesByIDs(ctx, uniqueInt64Keys(machineIDSet))
	if err != nil {
		return nil, err
	}

	routesByRevID := make(map[int64][]models.ProcessRouteDetail, len(headers))
	for _, header := range headers {
		ops := opByHeaderID[header.ID]
		if len(ops) == 0 {
			continue
		}
		mergedToolings := make([]models.RoutingOperationTooling, 0)
		for _, op := range ops {
			mergedToolings = append(mergedToolings, toolingsByOpID[op.ID]...)
		}
		revisionID, ok := headerRevisionMap[header.ID]
		if !ok {
			continue
		}
		routesByRevID[revisionID] = s.toRouteDetails(ops, mergedToolings, processNames, machineNames)
	}

	return &bomPreload{
		items:           itemMap,
		latestRevisions: latestRevisionMap,
		revisionByID:    revisionByID,
		assets:          assetMap,
		specs:           specMap,
		routesByRevID:   routesByRevID,
		children:        children,
	}, nil
}

func (p *bomPreload) assetByItemID(itemID int64) *models.ItemAsset {
	asset, ok := p.assets[itemID]
	if !ok {
		return nil
	}
	return &asset
}

func (p *bomPreload) childrenByParent(parentItemID int64, level int16, fallback []models.BomLine) []models.BomLine {
	if p != nil {
		if children, ok := p.children[lineTreeKey{parentItemID: parentItemID, level: level}]; ok {
			return children
		}
		return nil
	}

	rows := make([]models.BomLine, 0)
	for _, line := range fallback {
		if line.ParentItemID == parentItemID && line.Level == level {
			rows = append(rows, line)
		}
	}
	return rows
}

func (p *bomPreload) revisionForParent(bom models.BomItem) (models.ItemRevision, bool) {
	if p == nil {
		return models.ItemRevision{}, false
	}
	if bom.RootItemRevisionID != nil {
		rev, ok := p.revisionByID[*bom.RootItemRevisionID]
		if ok {
			return rev, true
		}
	}
	rev, ok := p.latestRevisions[bom.ItemID]
	return rev, ok
}

func (p *bomPreload) revisionForChild(line models.BomLine, childItemID int64) (models.ItemRevision, bool) {
	if p == nil {
		return models.ItemRevision{}, false
	}
	if line.ChildItemRevisionID != nil {
		rev, ok := p.revisionByID[*line.ChildItemRevisionID]
		if ok {
			return rev, true
		}
	}
	rev, ok := p.latestRevisions[childItemID]
	return rev, ok
}

func uniqueInt64Keys(values map[int64]struct{}) []int64 {
	result := make([]int64, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

// ---------------------------------------------------------------------------
// ApproveBom — multi-level approval state machine using approval_instances
// ---------------------------------------------------------------------------

func (s *service) ApproveBom(ctx context.Context, bomID int64, userID string, userRoles []string, req models.ApproveBomRequest) (*awmodels.ApprovalInstance, error) {
	instance, err := s.repo.GetApprovalInstanceByRef(ctx, "bom", "bom_item", bomID)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, apperror.NotFound("approval record tidak ditemukan for this BOM")
	}
	if instance.Status == "approved" || instance.Status == "rejected" {
		return nil, apperror.BadRequest(fmt.Sprintf("BOM is already %s", instance.Status))
	}

	wf, err := s.repo.GetApprovalWorkflowByID(ctx, instance.ApprovalWorkflowID)
	if err != nil {
		return nil, err
	}
	if wf == nil {
		return nil, apperror.BadRequest("active approval workflow for bom tidak ditemukan")
	}

	requiredRole := approval.LevelRole(wf, int16(instance.CurrentLevel))
	if requiredRole == "" {
		return nil, apperror.BadRequest(fmt.Sprintf("no role configured for approval level %d", instance.CurrentLevel))
	}
	if !approval.HasRole(userRoles, requiredRole) {
		return nil, apperror.Forbidden(fmt.Sprintf("level %d approval requires role '%s'", instance.CurrentLevel, requiredRole))
	}

	now := time.Now().UTC().Format(time.RFC3339)
	lvlIdx := instance.CurrentLevel - 1
	note := ""
	if req.Notes != nil {
		note = strings.TrimSpace(*req.Notes)
	}

	if req.Action == "reject" {
		instance.ApprovalProgress.Levels[lvlIdx].Status = "rejected"
		instance.ApprovalProgress.Levels[lvlIdx].ApprovedBy = userID
		instance.ApprovalProgress.Levels[lvlIdx].ApprovedAt = now
		instance.ApprovalProgress.Levels[lvlIdx].Note = note
		instance.Status = "rejected"

		if bom, _ := s.repo.GetBomByID(ctx, bomID); bom != nil {
			bom.Status = "Draft"
			_ = s.repo.UpdateBomItem(ctx, bom)
		}
	} else {
		instance.ApprovalProgress.Levels[lvlIdx].Status = "approved"
		instance.ApprovalProgress.Levels[lvlIdx].ApprovedBy = userID
		instance.ApprovalProgress.Levels[lvlIdx].ApprovedAt = now
		instance.ApprovalProgress.Levels[lvlIdx].Note = note

		if instance.CurrentLevel >= instance.MaxLevel {
			instance.Status = "approved"
			if bom, _ := s.repo.GetBomByID(ctx, bomID); bom != nil {
				bom.Status = "Active"
				_ = s.repo.UpdateBomItem(ctx, bom)
			}
			_ = s.repo.BulkActivateItemsByBomID(ctx, bomID)
		} else {
			instance.CurrentLevel++
		}
	}

	if err := s.repo.UpdateApprovalInstance(ctx, instance); err != nil {
		return nil, err
	}
	return instance, nil
}

const bomMaxRoutes = 7

var bomImportItemHeaders = func() []string {
	h := []string{
		"bom_group", "row_type", "uniq_code", "parent_uniq_code", "part_name", "part_number", "model", "uom", "level",
		"qty_per_uniq",
		"status", "description", "material_grade", "grade", "form",
		"width_mm", "thickness_mm", "length_mm", "diameter_mm", "weight_kg", "supplier_code", "customer_cycle", "kategori",
	}
	for n := 1; n <= bomMaxRoutes; n++ {
		s := fmt.Sprintf("%d", n)
		h = append(h,
			"op_seq_"+s, "process_code_"+s, "machine_number_"+s,
			"cycle_time_sec_"+s, "setup_time_min_"+s, "machine_stroke_"+s, "tooling_ref_"+s,
		)
	}
	return h
}()

// loadTemplateMasterData membaca master data (supplier, process, machine, uom)
// langsung dari DB setiap kali dipanggil, tanpa cache, sehingga template selalu
// merefleksikan kondisi master data saat ini.
func (s *service) loadTemplateMasterData(ctx context.Context) (*bulkimport.BomTemplateMasterData, error) {
	md := &bulkimport.BomTemplateMasterData{}

	suppliers, err := s.repo.ListAllSuppliers(ctx)
	if err != nil {
		return nil, apperror.InternalWrap("list suppliers for template", err)
	}
	for _, sp := range suppliers {
		md.Suppliers = append(md.Suppliers, bulkimport.RefRow{Code: sp.SupplierCode, Name: sp.SupplierName})
	}

	processes, err := s.repo.ListAllProcesses(ctx)
	if err != nil {
		return nil, apperror.InternalWrap("list processes for template", err)
	}
	for _, pr := range processes {
		md.Processes = append(md.Processes, bulkimport.RefRow{Code: pr.ProcessCode, Name: pr.ProcessName})
	}

	machines, err := s.repo.ListAllMachines(ctx)
	if err != nil {
		return nil, apperror.InternalWrap("list machines for template", err)
	}
	for _, m := range machines {
		md.Machines = append(md.Machines, bulkimport.RefRow{Code: m.MachineNumber, Name: m.MachineName})
	}

	uoms, err := s.repo.ListAllUoms(ctx)
	if err != nil {
		return nil, apperror.InternalWrap("list uoms for template", err)
	}
	for _, u := range uoms {
		md.Uoms = append(md.Uoms, bulkimport.RefRow{Code: u.Code, Name: u.Name})
	}

	return md, nil
}

// DownloadImportTemplate selalu membangun ulang template dari master data
// terbaru (real-time), tidak pernah menyajikan salinan yang di-cache.
func (s *service) DownloadImportTemplate(ctx context.Context) ([]byte, error) {
	md, err := s.loadTemplateMasterData(ctx)
	if err != nil {
		return nil, err
	}

	f, err := bulkimport.BuildBomTemplate(md)
	if err != nil {
		return nil, apperror.InternalWrap("build bom template", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, apperror.InternalWrap("write bom template", err)
	}
	return buf.Bytes(), nil
}

func (s *service) DownloadImportErrors(ctx context.Context, token string) ([]byte, error) {
	if strings.TrimSpace(token) == "" {
		return nil, apperror.BadRequest("invalid error token")
	}
	if s.errorStore == nil {
		return nil, apperror.NotFound("error file tidak ditemukan or expired")
	}
	data, err := s.errorStore.Get(token)
	if err != nil {
		return nil, apperror.InternalWrap("download error file", err)
	}
	if len(data) == 0 {
		return nil, apperror.NotFound("error file tidak ditemukan or expired")
	}
	return data, nil
}

func (s *service) ImportFromExcel(ctx context.Context, filePath, fileName, uploadedBy, requestID string) (bulkimport.BulkResult, error) {
	f, err := excelize.OpenFile(filePath)
	if err != nil {
		return bulkimport.BulkResult{}, apperror.BadRequest("failed to open excel file")
	}
	defer f.Close()

	itemRows, itemErrs, allGroups, err := s.parseItemRows(ctx, f)
	if err != nil {
		return bulkimport.BulkResult{}, err
	}

	// Resolve supplier codes → IDs (batch, unique codes only)
	supplierCache := make(map[string]string) // code → uuid string
	for i, row := range itemRows {
		if row.SupplierCode == "" {
			continue
		}
		if id, cached := supplierCache[row.SupplierCode]; cached {
			itemRows[i].SupplierID = &id
			continue
		}
		supplier, lookupErr := s.repo.FindSupplierByCode(ctx, row.SupplierCode)
		if lookupErr != nil {
			return bulkimport.BulkResult{}, lookupErr
		}
		if supplier == nil {
			continue // supplier_code tidak ditemukan → supplier_id tetap null
		}
		idStr := supplier.UUID.String()
		supplierCache[row.SupplierCode] = idStr
		itemRows[i].SupplierID = &idStr
	}

	routeRows, routeErrs, err := s.parseInlineRoutes(ctx, itemRows)
	if err != nil {
		return bulkimport.BulkResult{}, err
	}

	rowErrs := append(itemErrs, routeErrs...)
	invalidGroups := make(map[string]struct{})
	for _, e := range rowErrs {
		for _, r := range itemRows {
			if e.Sheet == "Items" && e.Row == r.SheetRow {
				invalidGroups[r.BomGroup] = struct{}{}
			}
		}
	}

	routesByUniq := make(map[string][]models.BomImportRouteRow)
	for _, r := range routeRows {
		routesByUniq[r.UniqCode] = append(routesByUniq[r.UniqCode], r)
	}
	for uniqCode := range routesByUniq {
		sort.Slice(routesByUniq[uniqCode], func(i, j int) bool {
			return routesByUniq[uniqCode][i].OpSeq < routesByUniq[uniqCode][j].OpSeq
		})
	}

	groups := make(map[string][]models.BomImportItemRow)
	for _, row := range itemRows {
		groups[row.BomGroup] = append(groups[row.BomGroup], row)
	}

	totalGroups := len(allGroups)
	successCount := 0

	for groupName, rows := range groups {
		if _, bad := invalidGroups[groupName]; bad {
			continue
		}

		req, buildErr := buildCreateBomRequest(rows, routesByUniq)
		if buildErr != nil {
			rootRow := rows[0]
			rowErrs = append(rowErrs, bulkimport.RowError{
				Sheet:   "Items",
				Row:     rootRow.SheetRow,
				Field:   "bom_group",
				Message: buildErr.Error(),
				RawData: rootRow.RawData,
			})
			continue
		}

		if _, err := s.CreateBom(ctx, req); err != nil {
			root := findGroupRoot(rows)
			if root != nil {
				rowErrs = append(rowErrs, bulkimport.RowError{
					Sheet:   "Items",
					Row:     root.SheetRow,
					Field:   "bom_group",
					Message: err.Error(),
					RawData: root.RawData,
				})
			}
			continue
		}
		successCount++
	}

	failedCount := totalGroups - successCount
	status := bulkimport.StatusSuccess
	if failedCount == totalGroups {
		status = bulkimport.StatusFailed
	} else if failedCount > 0 || len(rowErrs) > 0 {
		status = bulkimport.StatusPartial
	}

	result := bulkimport.BulkResult{
		Status:       status,
		Total:        totalGroups,
		SuccessCount: successCount,
		FailedCount:  failedCount,
		Errors:       mergeRowErrors(rowErrs),
	}

	// Bangun file error (kalau ada) supaya bisa ikut disimpan di history.
	var errorBytes []byte
	if len(result.Errors) > 0 {
		errMD, mdErr := s.loadTemplateMasterData(ctx)
		if mdErr != nil {
			return bulkimport.BulkResult{}, mdErr
		}
		errFile, err := bulkimport.GenerateBomErrorExcel(result.Errors, errMD)
		if err != nil {
			return bulkimport.BulkResult{}, apperror.InternalWrap("generate error excel", err)
		}
		defer errFile.Close()

		var b bytes.Buffer
		if _, err := errFile.WriteTo(&b); err != nil {
			return bulkimport.BulkResult{}, apperror.InternalWrap("write error excel", err)
		}
		errorBytes = b.Bytes()

		// In-memory token store dipertahankan (opsional, backward-compat).
		if s.errorStore == nil {
			store, err := bulkimport.NewFileStore("")
			if err != nil {
				return bulkimport.BulkResult{}, apperror.InternalWrap("init error store", err)
			}
			s.errorStore = store
		}
		token, err := s.errorStore.Save(errorBytes)
		if err != nil {
			return bulkimport.BulkResult{}, apperror.InternalWrap("save error excel", err)
		}
		result.ErrorToken = token
	}

	// --- Simpan history ke DB (untuk SEMUA upload: sukses / sebagian / gagal) ---
	histStatus := string(result.Status)
	if histStatus == string(bulkimport.StatusFailed) {
		histStatus = "error" // samakan dengan istilah frontend: success|partial|error
	}
	var sizeKb int
	if fi, statErr := os.Stat(filePath); statErr == nil {
		sizeKb = int(fi.Size() / 1024)
	}
	history := &models.BomImportHistory{
		FileName:      fileName,
		FileSizeKb:    sizeKb,
		RowCount:      result.Total,
		UploadedBy:    uploadedBy,
		Status:        histStatus,
		Summary:       fmt.Sprintf("%d berhasil, %d gagal dari %d BOM", result.SuccessCount, result.FailedCount, result.Total),
		ImportedCount: result.SuccessCount,
		FailedCount:   result.FailedCount,
		RequestID:     requestID,
		ErrorFile:     errorBytes, // nil kalau tidak ada error
		PreviewRows:   buildImportPreviewSnapshot(itemRows, rowErrs),
	}
	if err := s.repo.CreateImportHistory(ctx, history); err != nil {
		// Jangan gagalkan import hanya karena history gagal disimpan.
		fmt.Printf("gagal simpan bom import history: %v\n", err)
	}

	return result, nil
}

func (s *service) ListImportHistory(ctx context.Context, limit int) ([]models.BomImportHistory, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.repo.ListImportHistory(ctx, limit)
}

func (s *service) DownloadImportHistoryError(ctx context.Context, id string) ([]byte, error) {
	data, err := s.repo.GetImportHistoryErrorFile(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, apperror.NotFound("file error tidak ditemukan")
	}
	return data, nil
}

func buildImportPreviewSnapshot(
	itemRows []models.BomImportItemRow,
	rowErrs []bulkimport.RowError,
) datatypes.JSON {
	errByRow := make(map[int]string)
	for _, e := range rowErrs {
		if e.Sheet != "Items" {
			continue
		}
		if _, exists := errByRow[e.Row]; !exists {
			errByRow[e.Row] = e.Message
		}
	}

	type previewRow struct {
		Row            string  `json:"row"`
		Error          string  `json:"error"`
		BomGroup       string  `json:"Bom Group"`
		RowType        string  `json:"Row Type"`
		UniqCode       string  `json:"Uniq Code"`
		ParentUniqCode string  `json:"Parent Uniq Code"`
		PartName       string  `json:"Part Name"`
		PartNumber     string  `json:"Part Number"`
		Model          string  `json:"Model"`
		Uom            string  `json:"Uom"`
		Level          int16   `json:"Level"`
		QtyPerUniq     float64 `json:"Qty Per Uniq"`
		Status         string  `json:"Status"`
	}

	out := make([]previewRow, 0, len(itemRows))
	for _, r := range itemRows {
		label := r.UniqCode
		if label == "" {
			label = r.BomGroup
		}
		out = append(out, previewRow{
			Row:            label,
			Error:          errByRow[r.SheetRow],
			BomGroup:       r.BomGroup,
			RowType:        r.RowType,
			UniqCode:       r.UniqCode,
			ParentUniqCode: r.ParentUniqCode,
			PartName:       r.PartName,
			PartNumber:     r.PartNumber,
			Model:          r.Model,
			Uom:            r.Uom,
			Level:          r.Level,
			QtyPerUniq:     r.QtyPerUniq,
			Status:         r.Status,
		})
	}

	b, err := json.Marshal(out)
	if err != nil {
		return nil
	}
	return datatypes.JSON(b)
}

func (s *service) parseItemRows(ctx context.Context, f *excelize.File) ([]models.BomImportItemRow, []bulkimport.RowError, map[string]struct{}, error) {
	rows, err := f.GetRows("Items")
	if err != nil {
		return nil, nil, nil, apperror.BadRequest("sheet Items tidak ditemukan")
	}
	if len(rows) < 2 {
		return nil, nil, map[string]struct{}{}, nil
	}

	result := make([]models.BomImportItemRow, 0, len(rows)-1)
	errRows := make([]bulkimport.RowError, 0)
	allGroups := make(map[string]struct{})
	uniqSeen := make(map[string]int)
	headerIndex := mapImportHeaderIndex(rows[0])

	for i := 1; i < len(rows); i++ {
		raw := readImportRaw(rows[i], 1, len(rows[0])-1)
		sheetRow := i + 1

		row := models.BomImportItemRow{
			SheetRow:       sheetRow,
			RawData:        raw,
			BomGroup:       strings.TrimSpace(getImportValue(raw, headerIndex, "bom_group")),
			RowType:        strings.ToUpper(strings.TrimSpace(getImportValue(raw, headerIndex, "row_type"))),
			UniqCode:       strings.TrimSpace(getImportValue(raw, headerIndex, "uniq_code")),
			ParentUniqCode: strings.TrimSpace(getImportValue(raw, headerIndex, "parent_uniq_code")),
			PartName:       strings.TrimSpace(getImportValue(raw, headerIndex, "part_name")),
			PartNumber:     strings.TrimSpace(getImportValue(raw, headerIndex, "part_number")),
			Model:          strings.TrimSpace(getImportValue(raw, headerIndex, "model")),
			Uom:            strings.TrimSpace(getImportValue(raw, headerIndex, "uom")),
			Status:         strings.TrimSpace(getImportValue(raw, headerIndex, "status")),
			Description:    strings.TrimSpace(getImportValue(raw, headerIndex, "description")),
			MaterialGrade:  strings.TrimSpace(getImportValue(raw, headerIndex, "material_grade")),
			Grade:          strings.TrimSpace(getImportValue(raw, headerIndex, "grade")),
			Form:           strings.TrimSpace(getImportValue(raw, headerIndex, "form")),
			SupplierCode:   strings.TrimSpace(getImportValue(raw, headerIndex, "supplier_code")),
			QtyPerUniq:     1,
		}

		if row.BomGroup == "" && row.UniqCode == "" && row.PartName == "" {
			continue
		}
		if row.BomGroup != "" {
			allGroups[row.BomGroup] = struct{}{}
		}

		if row.BomGroup == "" {
			errRows = append(errRows, bulkimport.RowError{Sheet: "Items", Row: sheetRow, Field: "bom_group", Message: "wajib diisi", RawData: raw})
			continue
		}
		if row.RowType != "ROOT" && row.RowType != "CHILD" {
			errRows = append(errRows, bulkimport.RowError{Sheet: "Items", Row: sheetRow, Field: "row_type", Message: "harus ROOT atau CHILD", RawData: raw})
			continue
		}
		if row.UniqCode == "" {
			errRows = append(errRows, bulkimport.RowError{Sheet: "Items", Row: sheetRow, Field: "uniq_code", Message: "wajib diisi", RawData: raw})
			continue
		}
		// Duplicate uniq_code check hanya diberlakukan untuk baris ROOT (parent BoM
		// wajib unik). Untuk baris CHILD, uniq boleh diulang — satu child part
		// boleh dipakai kembali oleh beberapa parent yang berbeda di file upload
		// yang sama (backend `resolveOrCreateItem` akan me-reuse item existing).
		if row.RowType == "ROOT" {
			if prev, ok := uniqSeen[row.UniqCode]; ok {
				errRows = append(errRows, bulkimport.RowError{Sheet: "Items", Row: sheetRow, Field: "uniq_code", Message: fmt.Sprintf("duplikat dengan baris %d", prev), RawData: raw})
				continue
			}
			uniqSeen[row.UniqCode] = sheetRow
		}

		// ROOT item must not already exist — CreateBom always inserts a new item
		// (unlike children which use resolveOrCreateItem and tolerate existing rows).
		if row.RowType == "ROOT" {
			if existing, _ := s.repo.GetItemByUniq(ctx, row.UniqCode); existing != nil {
				errRows = append(errRows, bulkimport.RowError{Sheet: "Items", Row: sheetRow, Field: "uniq_code", Message: fmt.Sprintf("'%s' sudah ada di database", row.UniqCode), RawData: raw})
				continue
			}
		}

		if row.PartName == "" {
			errRows = append(errRows, bulkimport.RowError{Sheet: "Items", Row: sheetRow, Field: "part_name", Message: "wajib diisi", RawData: raw})
			continue
		}
		if row.Uom == "" {
			errRows = append(errRows, bulkimport.RowError{Sheet: "Items", Row: sheetRow, Field: "uom", Message: "wajib diisi", RawData: raw})
			continue
		}
		if row.Status == "" {
			row.Status = "Active"
		} else if row.Status != "Active" && row.Status != "Inactive" {
			errRows = append(errRows, bulkimport.RowError{Sheet: "Items", Row: sheetRow, Field: "status", Message: "harus Active atau Inactive", RawData: raw})
			continue
		}

		if row.RowType == "CHILD" {
			if row.ParentUniqCode == "" {
				errRows = append(errRows, bulkimport.RowError{Sheet: "Items", Row: sheetRow, Field: "parent_uniq_code", Message: "wajib diisi untuk CHILD", RawData: raw})
				continue
			}
			lvl, err := strconv.Atoi(strings.TrimSpace(getImportValue(raw, headerIndex, "level")))
			if err != nil || lvl < 1 || lvl > 4 {
				errRows = append(errRows, bulkimport.RowError{Sheet: "Items", Row: sheetRow, Field: "level", Message: "harus 1-4", RawData: raw})
				continue
			}
			if rawQPU := strings.TrimSpace(getImportValue(raw, headerIndex, "qty_per_uniq")); rawQPU != "" {
				qpu, err := strconv.ParseFloat(rawQPU, 64)
				if err != nil || qpu <= 0 {
					errRows = append(errRows, bulkimport.RowError{Sheet: "Items", Row: sheetRow, Field: "qty_per_uniq", Message: "harus angka > 0", RawData: raw})
					continue
				}
				row.QtyPerUniq = qpu
			}
			row.Level = int16(lvl)
		}

		// Normalize form case-insensitively to match DB CHECK constraint.
		// e.g. WIRE / wire / Wire → Wire  |  invalid value → error before hitting DB.
		if row.Form != "" {
			normalized, ok := map[string]string{
				"plate": "Plate", "coil": "Coil", "pipe": "Pipe",
				"rod": "Rod", "wire": "Wire", "other": "Other",
			}[strings.ToLower(row.Form)]
			if !ok {
				errRows = append(errRows, bulkimport.RowError{
					Sheet:   "Items",
					Row:     sheetRow,
					Field:   "form",
					Message: fmt.Sprintf("nilai form '%s' tidak dikenal. Gunakan salah satu: Plate, Coil, Pipe, Rod, Wire, Other", row.Form),
					RawData: raw,
				})
				continue
			}
			row.Form = normalized
		}

		row.WidthMM = parseOptionalFloat(getImportValue(raw, headerIndex, "width_mm"))
		row.ThicknessMM = parseOptionalFloat(getImportValue(raw, headerIndex, "thickness_mm"))
		row.LengthMM = parseOptionalFloat(getImportValue(raw, headerIndex, "length_mm"))
		row.DiameterMM = parseOptionalFloat(getImportValue(raw, headerIndex, "diameter_mm"))
		row.WeightKG = parseOptionalFloat(getImportValue(raw, headerIndex, "weight_kg"))
		row.CustomerCycle = strings.TrimSpace(getImportValue(raw, headerIndex, "customer_cycle"))

		if tm := strings.ToLower(strings.TrimSpace(getImportValue(raw, headerIndex, "kategori"))); tm != "" {
			switch tm {
			case "subcon", "raw", "indirect":
				row.TypeMaterial = tm
			default:
				errRows = append(errRows, bulkimport.RowError{
					Sheet:   "Items",
					Row:     sheetRow,
					Field:   "kategori",
					Message: fmt.Sprintf("nilai kategori '%s' tidak dikenal. Gunakan salah satu: subcon, raw, indirect", tm),
					RawData: raw,
				})
				continue
			}
		}

		row.ProcessCodes = make([]string, bomMaxRoutes)
		row.MachineNumbers = make([]string, bomMaxRoutes)
		row.OpSeqs = make([]string, bomMaxRoutes)
		row.CycleTimeSecs = make([]string, bomMaxRoutes)
		row.SetupTimeMins = make([]string, bomMaxRoutes)
		row.MachineStrokes = make([]string, bomMaxRoutes)
		row.ToolingRefs = make([]string, bomMaxRoutes)
		for n := 1; n <= bomMaxRoutes; n++ {
			s := fmt.Sprintf("%d", n)
			idx := n - 1
			row.ProcessCodes[idx] = strings.TrimSpace(getImportValue(raw, headerIndex, "process_code_"+s))
			row.MachineNumbers[idx] = strings.TrimSpace(getImportValue(raw, headerIndex, "machine_number_"+s))
			row.OpSeqs[idx] = strings.TrimSpace(getImportValue(raw, headerIndex, "op_seq_"+s))
			row.CycleTimeSecs[idx] = strings.TrimSpace(getImportValue(raw, headerIndex, "cycle_time_sec_"+s))
			row.SetupTimeMins[idx] = strings.TrimSpace(getImportValue(raw, headerIndex, "setup_time_min_"+s))
			row.MachineStrokes[idx] = strings.TrimSpace(getImportValue(raw, headerIndex, "machine_stroke_"+s))
			row.ToolingRefs[idx] = strings.TrimSpace(getImportValue(raw, headerIndex, "tooling_ref_"+s))
		}

		result = append(result, row)
	}

	return result, errRows, allGroups, nil
}

func (s *service) parseInlineRoutes(ctx context.Context, itemRows []models.BomImportItemRow) ([]models.BomImportRouteRow, []bulkimport.RowError, error) {
	result := make([]models.BomImportRouteRow, 0)
	errRows := make([]bulkimport.RowError, 0)

	for _, item := range itemRows {
		opSeqCounter := 0
		for i := 0; i < bomMaxRoutes; i++ {
			processCode := item.ProcessCodes[i]
			if processCode == "" {
				continue
			}

			processID, ok := s.repo.GetProcessIDByCode(ctx, processCode)
			if !ok {
				errRows = append(errRows, bulkimport.RowError{
					Sheet:   "Items",
					Row:     item.SheetRow,
					Field:   fmt.Sprintf("process_code_%d", i+1),
					Message: fmt.Sprintf("'%s' tidak ditemukan", processCode),
					RawData: item.RawData,
				})
				continue
			}

			opSeq := 0
			if item.OpSeqs[i] != "" {
				if seq, err := strconv.Atoi(item.OpSeqs[i]); err == nil && seq > 0 {
					opSeq = seq
					if seq > opSeqCounter {
						opSeqCounter = seq
					}
				}
			}
			if opSeq == 0 {
				opSeqCounter += 10
				opSeq = opSeqCounter
			}

			route := models.BomImportRouteRow{
				SheetRow:  item.SheetRow,
				RawData:   item.RawData,
				UniqCode:  item.UniqCode,
				OpSeq:     opSeq,
				ProcessID: processID,
			}

			if mn := item.MachineNumbers[i]; mn != "" {
				machineID, ok := s.repo.GetMachineIDByNumber(ctx, mn)
				if !ok {
					errRows = append(errRows, bulkimport.RowError{
						Sheet:   "Items",
						Row:     item.SheetRow,
						Field:   fmt.Sprintf("machine_number_%d", i+1),
						Message: fmt.Sprintf("'%s' tidak ditemukan", mn),
						RawData: item.RawData,
					})
					continue
				}
				route.MachineID = &machineID
			}
			route.CycleTimeSec = parseOptionalFloat(item.CycleTimeSecs[i])
			route.SetupTimeMin = parseOptionalFloat(item.SetupTimeMins[i])
			if v := item.MachineStrokes[i]; v != "" {
				route.MachineStroke = &v
			}
			if v := item.ToolingRefs[i]; v != "" {
				route.ToolingRef = &v
			}

			result = append(result, route)
		}
	}

	return result, errRows, nil
}

func buildCreateBomRequest(rows []models.BomImportItemRow, routesByUniq map[string][]models.BomImportRouteRow) (models.CreateBomRequest, error) {
	root := findGroupRoot(rows)
	if root == nil {
		return models.CreateBomRequest{}, fmt.Errorf("bom_group tidak punya ROOT")
	}

	byParent := make(map[string][]models.BomImportItemRow)
	uniqMap := make(map[string]models.BomImportItemRow)
	for _, r := range rows {
		uniqMap[r.UniqCode] = r
		if r.RowType == "CHILD" {
			byParent[r.ParentUniqCode] = append(byParent[r.ParentUniqCode], r)
		}
	}
	for _, r := range rows {
		if r.RowType == "CHILD" {
			if _, ok := uniqMap[r.ParentUniqCode]; !ok {
				return models.CreateBomRequest{}, fmt.Errorf("parent_uniq_code %s tidak ditemukan", r.ParentUniqCode)
			}
		}
	}

	children, err := buildChildren(root.UniqCode, 1, byParent, routesByUniq)
	if err != nil {
		return models.CreateBomRequest{}, err
	}

	req := models.CreateBomRequest{
		UniqCode:      root.UniqCode,
		PartName:      root.PartName,
		Uom:           root.Uom,
		Status:        root.Status,
		ProcessRoutes: toProcessInputs(routesByUniq[root.UniqCode]),
		MaterialSpec:  toMaterialSpec(root),
		Children:      children,
	}
	if root.PartNumber != "" {
		v := root.PartNumber
		req.PartNumber = &v
	}
	if root.Model != "" {
		v := root.Model
		req.Model = &v
	}
	if root.Description != "" {
		v := root.Description
		req.Description = &v
	}

	return req, nil
}

func buildChildren(parentUniq string, level int16, byParent map[string][]models.BomImportItemRow, routesByUniq map[string][]models.BomImportRouteRow) ([]models.ChildInput, error) {
	childrenRows := byParent[parentUniq]
	if len(childrenRows) == 0 {
		return nil, nil
	}

	res := make([]models.ChildInput, 0, len(childrenRows))
	for _, r := range childrenRows {
		if r.Level != level {
			return nil, fmt.Errorf("level child %s tidak sesuai parent", r.UniqCode)
		}
		nested, err := buildChildren(r.UniqCode, level+1, byParent, routesByUniq)
		if err != nil {
			return nil, err
		}

		uniq := r.UniqCode
		name := r.PartName
		uom := r.Uom

		child := models.ChildInput{
			UniqCode:      &uniq,
			PartName:      &name,
			Uom:           &uom,
			Level:         r.Level,
			QtyPerUniq:    r.QtyPerUniq,
			ProcessRoutes: toProcessInputs(routesByUniq[r.UniqCode]),
			MaterialSpec:  toMaterialSpec(&r),
			Children:      nested,
		}
		if r.PartNumber != "" {
			v := r.PartNumber
			child.PartNumber = &v
		}
		if r.Model != "" {
			v := r.Model
			child.Model = &v
		}
		res = append(res, child)
	}

	return res, nil
}

func toProcessInputs(routes []models.BomImportRouteRow) []models.ProcessRouteInput {
	if len(routes) == 0 {
		return nil
	}
	result := make([]models.ProcessRouteInput, 0, len(routes))
	for _, r := range routes {
		result = append(result, models.ProcessRouteInput{
			OpSeq:         r.OpSeq,
			ProcessID:     r.ProcessID,
			MachineID:     r.MachineID,
			CycleTimeSec:  r.CycleTimeSec,
			SetupTimeMin:  r.SetupTimeMin,
			MachineStroke: r.MachineStroke,
			ToolingRef:    r.ToolingRef,
		})
	}
	return result
}

func toMaterialSpec(row *models.BomImportItemRow) *models.MaterialSpecInput {
	if row == nil {
		return nil
	}
	hasAny := row.MaterialGrade != "" || row.Grade != "" || row.TypeMaterial != "" || row.Form != "" || row.WidthMM != nil || row.ThicknessMM != nil || row.LengthMM != nil || row.DiameterMM != nil || row.WeightKG != nil || row.SupplierID != nil || row.CustomerCycle != ""
	if !hasAny {
		return nil
	}
	ms := &models.MaterialSpecInput{
		WidthMm:     row.WidthMM,
		ThicknessMm: row.ThicknessMM,
		LengthMm:    row.LengthMM,
		DiameterMm:  row.DiameterMM,
		WeightKg:    row.WeightKG,
		SupplierID:  row.SupplierID,
	}
	if row.MaterialGrade != "" {
		v := row.MaterialGrade
		ms.MaterialGrade = &v
	}
	if row.Grade != "" {
		v := row.Grade
		ms.Grade = &v
	}
	if row.TypeMaterial != "" {
		v := row.TypeMaterial
		ms.TypeMaterial = &v
	}
	if row.Form != "" {
		v := row.Form
		ms.Form = &v
	}
	if row.CustomerCycle != "" {
		v := row.CustomerCycle
		ms.CustomerCycle = &v
	}
	return ms
}

func readImportRaw(row []string, start, count int) []string {
	raw := make([]string, count)
	for i := 0; i < count; i++ {
		idx := start + i
		if idx < len(row) {
			raw[i] = strings.TrimSpace(row[idx])
		}
	}
	return raw
}

func parseOptionalFloat(v string) *float64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return nil
	}
	return &f
}

func parseOptionalInt64(v string) *int64 {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	i, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return nil
	}
	return &i
}

func parseBoolLike(v string) (bool, bool) {
	s := strings.ToLower(strings.TrimSpace(v))
	switch s {
	case "true", "1", "yes", "y":
		return true, true
	case "false", "0", "no", "n":
		return false, true
	default:
		return false, false
	}
}

func findGroupRoot(rows []models.BomImportItemRow) *models.BomImportItemRow {
	for i := range rows {
		if rows[i].RowType == "ROOT" {
			return &rows[i]
		}
	}
	return nil
}

func mergeRowErrors(in []bulkimport.RowError) []bulkimport.RowError {
	if len(in) == 0 {
		return nil
	}
	type key struct {
		sheet string
		row   int
	}
	acc := make(map[key]bulkimport.RowError)
	order := make([]key, 0)

	for _, e := range in {
		k := key{sheet: e.Sheet, row: e.Row}
		if ex, ok := acc[k]; ok {
			ex.Message = ex.Message + "; " + e.Field + ": " + e.Message
			acc[k] = ex
			continue
		}
		msg := e.Message
		if e.Field != "" {
			msg = e.Field + ": " + e.Message
		}
		acc[k] = bulkimport.RowError{Sheet: e.Sheet, Row: e.Row, Field: e.Field, Message: msg, RawData: e.RawData}
		order = append(order, k)
	}

	out := make([]bulkimport.RowError, 0, len(order))
	for _, k := range order {
		out = append(out, acc[k])
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Sheet == out[j].Sheet {
			return out[i].Row < out[j].Row
		}
		return out[i].Sheet < out[j].Sheet
	})
	return out
}

func mapImportHeaderIndex(headerRow []string) map[string]int {
	idx := make(map[string]int, len(headerRow))
	for i := 1; i < len(headerRow); i++ {
		h := strings.ToLower(strings.TrimSpace(headerRow[i]))
		if h == "" {
			continue
		}
		idx[h] = i - 1
	}
	return idx
}

func getImportValue(raw []string, idx map[string]int, key string) string {
	pos, ok := idx[strings.ToLower(key)]
	if !ok || pos < 0 || pos >= len(raw) {
		return ""
	}
	return raw[pos]
}

func cleanupTempFile(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	_ = os.Remove(path)
}

// ---------------------------------------------------------------------------
// GetBomFull
// ---------------------------------------------------------------------------

func (s *service) GetBomFull(ctx context.Context, bomID int64) (*models.BomFullResponse, error) {
	bom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
		return nil, err
	}

	lines, err := s.repo.GetBomLinesByBomIDs(ctx, []int64{bom.ID})
	if err != nil {
		return nil, err
	}

	preload, err := s.preloadBomData(ctx, []models.BomItem{*bom}, lines)
	if err != nil {
		return nil, err
	}

	parent, ok := preload.items[bom.ItemID]
	if !ok {
		return nil, apperror.NotFound("item tidak ditemukan")
	}

	resp := &models.BomFullResponse{
		BomID:       bom.ID,
		BomVersion:  bom.Version,
		IsArchived:  !bom.IsCurrent,
		UniqCode:    parent.UniqCode,
		PartName:    parent.PartName,
		PartNumber:  parent.PartNumber,
		Model:       parent.Model,
		Uom:         parent.Uom,
		Status:      parent.Status,
		Description: bom.Description,
		Asset:       s.buildAssetInfo(preload.assetByItemID(parent.ID)),
	}

	if rev, ok := preload.revisionForParent(*bom); ok {
		if spec, ok := preload.specs[rev.ID]; ok {
			resp.MaterialSpec = s.toSpecDetail(&spec)
		}
		if routes, ok := preload.routesByRevID[rev.ID]; ok {
			resp.ProcessRoutes = routes
		}
	}

	resp.Children = s.buildFullChildTree(lines, preload, parent.UniqCode, parent.ID, 1)
	return resp, nil
}

func (s *service) buildFullChildTree(lines []models.BomLine, preload *bomPreload, parentUniqCode string, parentItemID int64, level int16) []models.BomFullChild {
	childLines := preload.childrenByParent(parentItemID, level, lines)
	rows := make([]models.BomFullChild, 0, len(childLines))
	for _, line := range childLines {
		child, ok := preload.items[line.ChildItemID]
		if !ok {
			continue
		}
		row := models.BomFullChild{
			ChildID:        child.ID,
			LineID:         line.ID,
			UniqCode:       child.UniqCode,
			ParentUniqCode: parentUniqCode,
			Level:          level,
			QtyPerUniq:     line.QtyPerUniq,
			ScrapFactor:    line.ScrapFactor,
			IsPhantom:      line.IsPhantom,
			PartName:       child.PartName,
			PartNumber:     child.PartNumber,
			Model:          child.Model,
			Uom:            child.Uom,
			Asset:          s.buildAssetInfo(preload.assetByItemID(child.ID)),
		}
		if rev, ok := preload.revisionForChild(line, child.ID); ok {
			if spec, ok := preload.specs[rev.ID]; ok {
				row.MaterialSpec = s.toSpecDetail(&spec)
			}
			if routes, ok := preload.routesByRevID[rev.ID]; ok {
				row.ProcessRoutes = routes
			}
		}
		if level < 6 {
			row.Children = s.buildFullChildTree(lines, preload, child.UniqCode, child.ID, level+1)
		}
		rows = append(rows, row)
	}
	return rows
}

// ---------------------------------------------------------------------------
// ReplaceBom
// ---------------------------------------------------------------------------

// uploadedAsset holds the saved file path and detected asset type for one upload_key.
type uploadedAsset struct {
	URL       string
	AssetType string
}

func (s *service) ReplaceBom(ctx context.Context, bomID int64, req models.ReplaceBomRequest, files map[string]*multipart.FileHeader, userID string) (*models.ReplaceBomResponse, error) {
	// --- Phase A: validate upload_key ↔ files ---
	allKeys := collectUploadKeys(req)
	for _, key := range allKeys {
		if _, ok := files[key]; !ok {
			return nil, apperror.BadRequest(fmt.Sprintf("missing file for upload_key '%s'", key))
		}
	}
	for key := range files {
		found := false
		for _, k := range allKeys {
			if k == key {
				found = true
				break
			}
		}
		if !found {
			return nil, apperror.BadRequest(fmt.Sprintf("unexpected upload key '%s' has no matching upload_key in payload", key))
		}
	}

	// --- Phase B: upload files in parallel (outside DB) ---
	uploadedMap := make(map[string]uploadedAsset, len(files))
	if len(files) > 0 {
		type uploadResult struct {
			key   string
			asset uploadedAsset
			err   error
		}
		results := make(chan uploadResult, len(files))
		tasks := make([]concurrency.Task, 0, len(files))
		for k, fh := range files {
			k, fh := k, fh
			tasks = append(tasks, func(ctx context.Context) error {
				url, assetType, err := saveTempFile(fh)
				results <- uploadResult{key: k, asset: uploadedAsset{URL: url, AssetType: assetType}, err: err}
				return err
			})
		}
		if err := concurrency.Run(ctx, tasks, concurrency.DefaultFanout); err != nil {
			close(results)
			return nil, apperror.BadRequest(fmt.Sprintf("asset upload failed: %v", err))
		}
		close(results)
		for r := range results {
			uploadedMap[r.key] = r.asset
		}
	}

	// --- Phase C: load and validate old BOM (reads only, outside transaction) ---
	oldBom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
		return nil, err
	}

	versions, err := s.repo.GetBomVersionsByItemID(ctx, oldBom.ItemID)
	if err != nil {
		return nil, err
	}
	nextVersion := oldBom.Version + 1
	for _, v := range versions {
		if v.Version >= nextVersion {
			nextVersion = v.Version + 1
		}
	}

	parentItem, err := s.repo.GetItemByID(ctx, oldBom.ItemID)
	if err != nil {
		return nil, err
	}

	// --- Phases D–G: all DB writes inside a single transaction ---
	var result models.ReplaceBomResponse
	txErr := s.repo.DB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		txSvc := &service{repo: repository.New(tx), errorStore: s.errorStore}

		// Phase D: new item_revision for parent
		newParentRev, err := txSvc.createNextItemRevision(ctx, parentItem.ID, nextVersion, req.ChangeNote)
		if err != nil {
			return err
		}

		parentItem.PartName = req.PartName
		if req.PartNumber != nil {
			parentItem.PartNumber = req.PartNumber
		}
		if req.Model != nil {
			parentItem.Model = req.Model
		}
		parentItem.Uom = req.Uom
		revLabel := newParentRev.Revision
		parentItem.CurrentRevision = &revLabel
		if err := txSvc.repo.UpdateItem(ctx, parentItem); err != nil {
			return err
		}

		assetsReused, assetsUploaded := 0, 0
		if req.UploadKey != nil {
			if ua, ok := uploadedMap[*req.UploadKey]; ok {
				if err := txSvc.repo.UpsertItemAssetURL(ctx, parentItem.ID, ua.AssetType, ua.URL); err != nil {
					return err
				}
				assetsUploaded++
			}
		} else if req.AssetID != nil {
			assetsReused++
		}

		if len(req.ProcessRoutes) > 0 {
			if err := txSvc.createRouting(ctx, parentItem.ID, newParentRev.ID, req.ProcessRoutes); err != nil {
				return err
			}
		}

		if req.MaterialSpec != nil {
			if err := txSvc.saveMaterialSpec(ctx, newParentRev.ID, req.MaterialSpec); err != nil {
				return err
			}
		}

		// Phase E: deactivate old versions, create new bom_item
		for i := range versions {
			if versions[i].IsCurrent {
				versions[i].IsCurrent = false
				if err := txSvc.repo.UpdateBomItem(ctx, &versions[i]); err != nil {
					return err
				}
			}
		}

		newBom := &models.BomItem{
			ItemID:             parentItem.ID,
			RootItemRevisionID: &newParentRev.ID,
			CopiedFromBomID:    &oldBom.ID,
			Version:            nextVersion,
			Status:             "Released",
			Description:        req.Description,
			ChangeNote:         req.ChangeNote,
			IsCurrent:          true,
		}
		if err := txSvc.repo.CreateBomItem(ctx, newBom); err != nil {
			return err
		}

		// Phase F: process children recursively
		ru, up, err := txSvc.replaceChildren(ctx, newBom.ID, parentItem.UniqCode, parentItem.ID, req.Children, uploadedMap)
		if err != nil {
			return err
		}
		assetsReused += ru
		assetsUploaded += up

		// Phase G: create approval instance
		wf, err := txSvc.repo.GetApprovalWorkflowByActionName(ctx, "bom")
		if err != nil {
			return err
		}
		if wf == nil {
			return apperror.BadRequest("no active approval workflow configured for action 'bom'")
		}
		maxLevel := approval.MaxLevel(wf)
		if maxLevel < 1 {
			return apperror.BadRequest("no approval levels configured for workflow 'bom'")
		}
		instance := &awmodels.ApprovalInstance{
			ActionName:         "bom",
			ReferenceTable:     "bom_item",
			ReferenceID:        newBom.ID,
			ApprovalWorkflowID: wf.ID,
			CurrentLevel:       1,
			MaxLevel:           maxLevel,
			Status:             "pending",
			SubmittedBy:        userID,
			ApprovalProgress:   approval.BuildProgress(wf, maxLevel),
		}
		if err := txSvc.repo.CreateApprovalInstance(ctx, instance); err != nil {
			return err
		}

		result = models.ReplaceBomResponse{
			NewBomID:       newBom.ID,
			OldBomID:       oldBom.ID,
			UniqCode:       parentItem.UniqCode,
			NewBomVersion:  newBom.Version,
			AssetsReused:   assetsReused,
			AssetsUploaded: assetsUploaded,
		}
		return nil
	})
	if txErr != nil {
		return nil, txErr
	}

	return &result, nil
}

// replaceChildren recursively processes children payload: resolves/creates items,
// handles revisions, routing, material spec, asset, and creates bom_lines.
// Returns (assetsReused, assetsUploaded, error).
func (s *service) replaceChildren(
	ctx context.Context,
	bomID int64,
	parentUniqCode string,
	parentItemID int64,
	children []models.ReplaceBomChildInput,
	uploadedMap map[string]uploadedAsset,
) (int, int, error) {
	reused, uploaded := 0, 0
	for _, c := range children {
		// Resolve or create item
		childItem, err := s.resolveOrCreateReplaceItem(ctx, c)
		if err != nil {
			return reused, uploaded, err
		}

		// New revision for this child
		childRev, err := s.createNextItemRevision(ctx, childItem.ID, 1, nil)
		if err != nil {
			return reused, uploaded, err
		}
		revLabel := childRev.Revision
		childItem.CurrentRevision = &revLabel
		_ = s.repo.UpdateItem(ctx, childItem)

		// Asset handling
		if c.UploadKey != nil {
			if ua, ok := uploadedMap[*c.UploadKey]; ok {
				if err := s.repo.UpsertItemAssetURL(ctx, childItem.ID, ua.AssetType, ua.URL); err != nil {
					return reused, uploaded, err
				}
				uploaded++
			}
		} else if c.AssetID != nil {
			reused++
		}

		// Routing
		if len(c.ProcessRoutes) > 0 {
			if err := s.createRouting(ctx, childItem.ID, childRev.ID, c.ProcessRoutes); err != nil {
				return reused, uploaded, err
			}
		}

		// Material spec
		if c.MaterialSpec != nil {
			if err := s.saveMaterialSpec(ctx, childRev.ID, c.MaterialSpec); err != nil {
				return reused, uploaded, err
			}
		}

		// bom_line
		line := &models.BomLine{
			BomItemID:           bomID,
			ParentItemID:        parentItemID,
			ChildItemID:         childItem.ID,
			ChildItemRevisionID: &childRev.ID,
			Level:               c.Level,
			QtyPerUniq:          c.QtyPerUniq,
			ScrapFactor:         c.ScrapFactor,
			IsPhantom:           c.IsPhantom,
		}
		if c.Uom != "" {
			uom := c.Uom
			line.Uom = &uom
		}
		if err := s.repo.CreateBomLine(ctx, line); err != nil {
			return reused, uploaded, err
		}

		// Recurse
		if len(c.Children) > 0 && c.Level < 6 {
			ru, up, err := s.replaceChildren(ctx, bomID, c.UniqCode, childItem.ID, c.Children, uploadedMap)
			if err != nil {
				return reused, uploaded, err
			}
			reused += ru
			uploaded += up
		}
	}
	return reused, uploaded, nil
}

// resolveOrCreateReplaceItem finds an existing item by uniq_code or creates a new one.
func (s *service) resolveOrCreateReplaceItem(ctx context.Context, c models.ReplaceBomChildInput) (*models.Item, error) {
	existing, err := s.repo.GetItemByUniq(ctx, c.UniqCode)
	if err == nil && existing != nil {
		// Update mutable fields
		existing.PartName = c.PartName
		if c.PartNumber != nil {
			existing.PartNumber = c.PartNumber
		}
		if c.Model != nil {
			existing.Model = c.Model
		}
		existing.Uom = c.Uom
		_ = s.repo.UpdateItem(ctx, existing)
		return existing, nil
	}

	// Create new item
	item := &models.Item{
		UniqCode:   c.UniqCode,
		PartName:   c.PartName,
		PartNumber: c.PartNumber,
		Model:      c.Model,
		Uom:        c.Uom,
		Status:     "Draft",
	}
	if err := s.repo.CreateItem(ctx, item); err != nil {
		return nil, err
	}
	return item, nil
}

// collectUploadKeys walks the full request tree and collects all non-nil upload_keys.
func collectUploadKeys(req models.ReplaceBomRequest) []string {
	keys := make([]string, 0)
	if req.UploadKey != nil && *req.UploadKey != "" {
		keys = append(keys, *req.UploadKey)
	}
	keys = append(keys, collectChildUploadKeys(req.Children)...)
	return keys
}

func collectChildUploadKeys(children []models.ReplaceBomChildInput) []string {
	keys := make([]string, 0)
	for _, c := range children {
		if c.UploadKey != nil && *c.UploadKey != "" {
			keys = append(keys, *c.UploadKey)
		}
		keys = append(keys, collectChildUploadKeys(c.Children)...)
	}
	return keys
}

// saveTempFile writes a multipart file to uploads/assets/replace/ and returns (url, assetType, error).
func saveTempFile(fh *multipart.FileHeader) (string, string, error) {
	const dir = "uploads/assets/replace"
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", "", fmt.Errorf("mkdir replace dir: %w", err)
	}

	src, err := fh.Open()
	if err != nil {
		return "", "", fmt.Errorf("open upload: %w", err)
	}
	defer src.Close()

	filename := fmt.Sprintf("%s_%s", uuid.New().String()[:8], filepath.Base(fh.Filename))
	finalPath := filepath.Join(dir, filename)

	dst, err := os.Create(finalPath)
	if err != nil {
		return "", "", fmt.Errorf("create file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", "", fmt.Errorf("write file: %w", err)
	}

	url := "/" + filepath.ToSlash(finalPath)
	return url, detectAssetType(fh.Filename), nil
}

// detectAssetType returns the asset_type string based on file extension.
func detectAssetType(filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".glb", ".gltf", ".obj", ".fbx", ".stl":
		return "3d-model"
	case ".dxf", ".dwg", ".pdf":
		return "drawing"
	case ".png", ".jpg", ".jpeg", ".webp", ".gif":
		return "photo"
	default:
		return "other"
	}
}
