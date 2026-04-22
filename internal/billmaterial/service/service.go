// Package service implements business logic for the Bill of Material module.
package service

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	//"time"

	awmodels "github.com/ganasa18/go-template/internal/approval_workflow/models"
	"github.com/ganasa18/go-template/internal/billmaterial/models"
	"github.com/ganasa18/go-template/internal/billmaterial/repository"
	"github.com/ganasa18/go-template/pkg/apperror"

	"github.com/ganasa18/go-template/pkg/approval"
	"github.com/ganasa18/go-template/pkg/pagination"
	"github.com/google/uuid"
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
	//ApproveBom(ctx context.Context, bomID int64, userID string, userRoles []string, req models.ApproveBomRequest) (*awmodels.ApprovalInstance, error)
}

type service struct{ repo repository.IRepository }

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

func New(repo repository.IRepository) IService { return &service{repo: repo} }

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
		UniqCode:       q.UniqCode,
		Status:         q.Status,
		Search:         q.Search,
		Page:           page,
		Limit:          limit,
		OrderBy:        q.OrderBy,
		OrderDirection: q.OrderDirection,
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
			Level:      "Parent",
			Asset:      s.buildAssetInfo(preload.assetByItemID(parent.ID)),
			Status:     parent.Status,
		}
		if parentRev, ok := preload.revisionForParent(b); ok {
			row.Version = &parentRev.Revision
		}

		row.Children = s.buildChildTree(linesByBomID[b.ID], preload, parent.ID, 1)

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
func (s *service) buildChildTree(lines []models.BomLine, preload *bomPreload, parentItemID int64, level int16) []models.BomTreeRow {
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
			Level:      int(level),
			QPU:        &qpu,
			Asset:      s.buildAssetInfo(preload.assetByItemID(child.ID)),
			Status:     child.Status,
		}
		if rev, ok := preload.revisionForChild(line, child.ID); ok {
			row.Version = &rev.Revision
		}
		if level < 4 {
			row.Children = s.buildChildTree(lines, preload, child.ID, level+1)
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
	if err := s.repo.CreateItem(ctx, parent); err != nil {
		return nil, err
	}

	// 2. Create default revision
	revStr := "v1"
	parent.CurrentRevision = &revStr
	_ = s.repo.UpdateItem(ctx, parent)
	rev := &models.ItemRevision{
		ItemID:     parent.ID,
		Revision:   revStr,
		Status:     "Draft",
		ChangeNote: req.Description,
	}
	if err := s.repo.CreateRevision(ctx, rev); err != nil {
		return nil, err
	}

	// 3. Picture
	if req.PictureURL != nil {
		_ = s.repo.CreateAsset(ctx, &models.ItemAsset{
			ItemID:    parent.ID,
			AssetType: "photo",
			FileURL:   *req.PictureURL,
			Status:    "Active",
		})
	}

	// 4. Process routes
	if len(req.ProcessRoutes) > 0 {
		if err := s.createRouting(ctx, parent.ID, rev.ID, req.ProcessRoutes); err != nil {
			return nil, err
		}
	}

	// 5. Material spec
	if req.MaterialSpec != nil {
		if err := s.saveMaterialSpec(ctx, rev.ID, req.MaterialSpec); err != nil {
			return nil, err
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
	if err := s.repo.CreateBomItem(ctx, bom); err != nil {
		return nil, err
	}

	// 6a. Auto-create approval instance if workflow is configured.
	wf, err := s.repo.GetApprovalWorkflowByActionName(ctx, "bom")
	if err != nil {
		return nil, err
	}
	if wf != nil {
		maxLevel := approval.MaxLevel(wf)
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
		if err := s.repo.CreateApprovalInstance(ctx, instance); err != nil {
			return nil, err
		}
	}

	// 7. Recurse children
	if err := s.createChildren(ctx, bom.ID, parent.ID, req.Children); err != nil {
		return nil, err
	}

	return s.GetBomDetail(ctx, bom.ID)
}

// createChildren resolves or creates each child item and the bom_line, then recurses.
func (s *service) createChildren(ctx context.Context, bomID, parentItemID int64, children []models.ChildInput) error {
	for _, c := range children {
		childID, childRevID, err := s.resolveOrCreateItem(ctx, c)
		if err != nil {
			return err
		}
		if childID == parentItemID {
			return apperror.BadRequest("child cannot be the same as parent")
		}

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

		if len(c.Children) > 0 && c.Level < 4 {
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
	for i, pr := range routes {
		seq, ok := seqMap[pr.ProcessID]
		if !ok {
			return apperror.BadRequest(fmt.Sprintf("process_id %d does not exist (op_seq %d)", pr.ProcessID, pr.OpSeq))
		}
		if seq < prevSeq {
			return apperror.BadRequest(
				fmt.Sprintf("route index %d (process_id %d) has master sequence %d which is smaller than the previous step sequence %d — routing must follow ascending process order",
					i, pr.ProcessID, seq, prevSeq),
			)
		}
		prevSeq = seq
	}

	rh := &models.RoutingHeader{ItemID: itemID, ItemRevisionID: &revID, Version: 1, Status: "Draft"}
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
		Form:           ms.Form,
		WidthMm:        ms.WidthMm,
		DiameterMm:     ms.DiameterMm,
		ThicknessMm:    ms.ThicknessMm,
		LengthMm:       ms.LengthMm,
		WeightKg:       ms.WeightKg,
		CycleTimeSec:   ms.CycleTimeSec,
		SetupTimeMin:   ms.SetupTimeMin,
	}
	if ms.SupplierID != nil {
		if _, err := uuid.Parse(*ms.SupplierID); err != nil {
			return apperror.BadRequest("invalid supplier_id")
		}
		spec.SupplierID = ms.SupplierID
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
		return nil, apperror.NotFound("item not found")
	}

	resp := &models.BomDetailResponse{
		BomID:       bom.ID,
		BomVersion:  bom.Version,
		BomStatus:   bom.Status,
		IsCurrent:   bom.IsCurrent,
		ReadOnly:    bom.Status != "Draft",
		ChangeNote:  bom.ChangeNote,
		ID:          parent.ID,
		UniqCode:    parent.UniqCode,
		PartName:    parent.PartName,
		PartNumber:  parent.PartNumber,
		Model:       parent.Model,
		Status:      parent.Status,
		Description: bom.Description,
		Asset:       s.buildAssetInfo(preload.assetByItemID(parent.ID)),
	}
	if parentRev, ok := preload.revisionForParent(*bom); ok {
		resp.Version = &parentRev.Revision
		if spec, ok := preload.specs[parentRev.ID]; ok {
			resp.MaterialSpec = s.toSpecDetail(&spec)
		}
	}

	if parentRev, ok := preload.revisionForParent(*bom); ok {
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
		return nil, apperror.NotFound("source item revision not found")
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
		return nil, apperror.NotFound("source item revision not found")
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
		return nil, apperror.NotFound("routing operation not found in bom version")
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
			return 0, 0, nil, apperror.NotFound("root item revision not found")
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
		return 0, 0, nil, apperror.NotFound("child item revision not found")
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
	if _, err := s.repo.GetBomByID(ctx, bomID); err != nil {
		return err
	}
	return s.repo.DeleteBomItem(ctx, bomID)
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
		return 0, apperror.NotFound("child item not found in bom")
	}

	lineIDs := collectSubtreeLineIDs(lines, roots)
	deleted, err := s.repo.DeleteBomLinesByIDs(ctx, bomID, lineIDs)
	if err != nil {
		return 0, err
	}
	if deleted == 0 {
		return 0, apperror.NotFound("child item not found in bom")
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
		return 0, apperror.NotFound("line not found in bom")
	}

	lineIDs := collectSubtreeLineIDs(lines, []models.BomLine{*root})
	deleted, err := s.repo.DeleteBomLinesByIDs(ctx, bomID, lineIDs)
	if err != nil {
		return 0, err
	}
	if deleted == 0 {
		return 0, apperror.NotFound("line not found in bom")
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
			ID:         child.ID,
			LineID:     line.ID,
			UniqCode:   child.UniqCode,
			PartName:   child.PartName,
			PartNumber: child.PartNumber,
			Model:      child.Model,
			Level:      level,
			QPU:        line.QtyPerUniq,
			Asset:      s.buildAssetInfo(preload.assetByItemID(child.ID)),
			Status:     child.Status,
		}
		if rev, ok := preload.revisionForChild(line, child.ID); ok {
			row.Version = &rev.Revision
			if spec, ok := preload.specs[rev.ID]; ok {
				row.MaterialSpec = s.toSpecDetail(&spec)
			}
		}
		if rev, ok := preload.revisionForChild(line, child.ID); ok {
			if routes, ok := preload.routesByRevID[rev.ID]; ok {
				row.ProcessRoutes = routes
			}
		}
		if level < 4 {
			row.Children = s.buildDetailTree(lines, preload, child.ID, level+1)
		}
		rows = append(rows, row)
	}
	return rows
}

func (s *service) toSpecDetail(spec *models.ItemMaterialSpec) *models.MaterialSpecDetail {
	d := &models.MaterialSpecDetail{
		MaterialGrade: spec.MaterialGrade,
		Form:          spec.Form,
		WidthMm:       spec.WidthMm,
		DiameterMm:    spec.DiameterMm,
		ThicknessMm:   spec.ThicknessMm,
		LengthMm:      spec.LengthMm,
		WeightKg:      spec.WeightKg,
		CycleTimeSec:  spec.CycleTimeSec,
		SetupTimeMin:  spec.SetupTimeMin,
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

	headers := make([]models.RoutingHeader, 0, len(revisionByID))
	for revisionID := range revisionByID {
		header, err := s.repo.GetRoutingHeaderByRevisionID(ctx, revisionID)
		if err != nil {
			return nil, err
		}
		if header != nil {
			headers = append(headers, *header)
		}
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

// func (s *service) ApproveBom(ctx context.Context, bomID int64, userID string, userRoles []string, req models.ApproveBomRequest) (*awmodels.ApprovalInstance, error) {
// 	instance, err := s.repo.GetApprovalInstanceByRef(ctx, "bom", "bom_item", bomID)
// 	if err != nil {
// 		return nil, err
// 	}
// 	if instance == nil {
// 		return nil, apperror.NotFound("approval record not found for this BOM")
// 	}
// 	if instance.Status == "approved" || instance.Status == "rejected" {
// 		return nil, apperror.BadRequest(fmt.Sprintf("BOM is already %s", instance.Status))
// 	}

// 	wf, err := s.repo.GetApprovalWorkflowByID(ctx, instance.ApprovalWorkflowID)
// 	if err != nil {
// 		return nil, err
// 	}

// 	requiredRole := approval.LevelRole(wf, int16(instance.CurrentLevel))
// 	if requiredRole == "" {
// 		return nil, apperror.BadRequest(fmt.Sprintf("no role configured for approval level %d", instance.CurrentLevel))
// 	}
// 	if !approval.HasRole(userRoles, requiredRole) {
// 		return nil, apperror.Forbidden(fmt.Sprintf(
// 			"level %d approval requires role '%s'", instance.CurrentLevel, requiredRole,
// 		))
// 	}

// 	now := time.Now().UTC().Format(time.RFC3339)
// 	lvlIdx := instance.CurrentLevel - 1 // 0-based index into Levels slice
// 	note := ""
// 	if req.Notes != nil {
// 		note = *req.Notes
// 	}

// 	if req.Action == "reject" {
// 		instance.ApprovalProgress.Levels[lvlIdx].Status = "rejected"
// 		instance.ApprovalProgress.Levels[lvlIdx].ApprovedBy = userID
// 		instance.ApprovalProgress.Levels[lvlIdx].ApprovedAt = now
// 		instance.ApprovalProgress.Levels[lvlIdx].Note = note
// 		instance.Status = "rejected"

// 		// Reset bom_item back to Draft so creator can revise and resubmit
// 		if bom, _ := s.repo.GetBomByID(ctx, bomID); bom != nil {
// 			bom.Status = "Draft"
// 			_ = s.repo.UpdateBomItem(ctx, bom)
// 		}
// 	} else {
// 		instance.ApprovalProgress.Levels[lvlIdx].Status = "approved"
// 		instance.ApprovalProgress.Levels[lvlIdx].ApprovedBy = userID
// 		instance.ApprovalProgress.Levels[lvlIdx].ApprovedAt = now
// 		instance.ApprovalProgress.Levels[lvlIdx].Note = note

// 		if instance.CurrentLevel >= instance.MaxLevel {
// 			// All levels passed — activate bom_item + all children items
// 			instance.Status = "approved"
// 			if bom, _ := s.repo.GetBomByID(ctx, bomID); bom != nil {
// 				bom.Status = "Active"
// 				_ = s.repo.UpdateBomItem(ctx, bom)
// 			}
// 			_ = s.repo.BulkActivateItemsByBomID(ctx, bomID)
// 		} else {
// 			instance.CurrentLevel++
// 		}
// 	}

// 	if err := s.repo.UpdateApprovalInstance(ctx, instance); err != nil {
// 		return nil, err
// 	}
// 	return instance, nil
// }
