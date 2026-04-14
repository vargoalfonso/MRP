// Package service implements business logic for the Bill of Material module.
package service

import (
	"context"
	"fmt"
	"sort"

	"github.com/ganasa18/go-template/internal/billmaterial/models"
	"github.com/ganasa18/go-template/internal/billmaterial/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
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
}

type service struct{ repo repository.IRepository }

type lineTreeKey struct {
	parentItemID int64
	level        int16
}

type bomPreload struct {
	items     map[int64]models.Item
	revisions map[int64]models.ItemRevision
	assets    map[int64]models.ItemAsset
	specs     map[int64]models.ItemMaterialSpec
	routes    map[int64][]models.ProcessRouteDetail
	children  map[lineTreeKey][]models.BomLine
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
		row := models.BomTreeRow{
			ID:         parent.ID,
			UniqCode:   parent.UniqCode,
			PartName:   parent.PartName,
			PartNumber: parent.PartNumber,
			Level:      "Parent",
			Asset:      s.buildAssetInfo(preload.assetByItemID(parent.ID)),
			Status:     parent.Status,
		}
		if parentRev, ok := preload.revisions[parent.ID]; ok {
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
			Level:      int(level),
			QPU:        &qpu,
			Asset:      s.buildAssetInfo(preload.assetByItemID(child.ID)),
			Status:     child.Status,
		}
		if rev, ok := preload.revisions[child.ID]; ok {
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
	status := "Active"
	if req.Status != "" {
		status = req.Status
	}
	parent := &models.Item{
		UniqCode:   req.UniqCode,
		PartName:   req.PartName,
		PartNumber: req.PartNumber,
		Uom:        req.Uom,
		Status:     status,
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
		ItemID:      parent.ID,
		Version:     1,
		Status:      "Draft",
		Description: req.Description,
	}
	if err := s.repo.CreateBomItem(ctx, bom); err != nil {
		return nil, err
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
		childID, err := s.resolveOrCreateItem(ctx, c)
		if err != nil {
			return err
		}
		if childID == parentItemID {
			return apperror.BadRequest("child cannot be the same as parent")
		}

		line := &models.BomLine{
			BomItemID:    bomID,
			ParentItemID: parentItemID,
			ChildItemID:  childID,
			Level:        c.Level,
			QtyPerUniq:   c.QtyPerUniq,
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
func (s *service) resolveOrCreateItem(ctx context.Context, c models.ChildInput) (int64, error) {
	if c.ItemID != nil {
		if _, err := s.repo.GetItemByID(ctx, *c.ItemID); err != nil {
			return 0, err
		}
		return *c.ItemID, nil
	}

	if c.UniqCode == nil || c.PartName == nil {
		return 0, apperror.BadRequest("child must have item_id or uniq_code + part_name")
	}
	if c.Uom == nil {
		return 0, apperror.BadRequest("child requires uom when creating new item: " + *c.UniqCode)
	}

	item := &models.Item{
		UniqCode:   *c.UniqCode,
		PartName:   *c.PartName,
		PartNumber: c.PartNumber,
		Uom:        *c.Uom,
		Status:     "Active",
	}
	if err := s.repo.CreateItem(ctx, item); err != nil {
		return 0, err
	}

	revStr := "v1"
	if c.Revision != nil {
		revStr = *c.Revision
	}
	item.CurrentRevision = &revStr
	_ = s.repo.UpdateItem(ctx, item)

	rev := &models.ItemRevision{ItemID: item.ID, Revision: revStr, Status: "Draft"}
	if err := s.repo.CreateRevision(ctx, rev); err != nil {
		return 0, err
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

	return item.ID, nil
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
		return nil, apperror.NotFound("item not found")
	}

	resp := &models.BomDetailResponse{
		BomID:       bom.ID,
		BomVersion:  bom.Version,
		BomStatus:   bom.Status,
		ID:          parent.ID,
		UniqCode:    parent.UniqCode,
		PartName:    parent.PartName,
		PartNumber:  parent.PartNumber,
		Status:      parent.Status,
		Description: bom.Description,
		Asset:       s.buildAssetInfo(preload.assetByItemID(parent.ID)),
	}
	if parentRev, ok := preload.revisions[parent.ID]; ok {
		resp.Version = &parentRev.Revision
		if spec, ok := preload.specs[parentRev.ID]; ok {
			resp.MaterialSpec = s.toSpecDetail(&spec)
		}
	}

	if routes, ok := preload.routes[parent.ID]; ok {
		resp.ProcessRoutes = routes
	}

	resp.Children = s.buildDetailTree(lines, preload, parent.ID, 1)

	return resp, nil
}

func (s *service) UpdateBom(ctx context.Context, bomID int64, req models.UpdateBomRequest) (*models.BomDetailResponse, error) {
	bom, err := s.repo.GetBomByID(ctx, bomID)
	if err != nil {
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
		if err := s.repo.DeleteRoutingByItemID(ctx, item.ID); err != nil {
			return nil, err
		}
		if len(*req.ProcessRoutes) > 0 {
			rev, err := s.repo.GetLatestRevision(ctx, item.ID)
			if err != nil {
				return nil, err
			}
			var revID int64
			if rev != nil {
				revID = rev.ID
			}
			if err := s.createRouting(ctx, item.ID, revID, *req.ProcessRoutes); err != nil {
				return nil, err
			}
		}
	}

	// Upsert material spec when provided
	if req.MaterialSpec != nil {
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

	return s.GetBomDetail(ctx, bomID)
}

func (s *service) UpdateBomChild(ctx context.Context, bomID, lineID int64, req models.UpdateBomChildRequest) (*models.BomDetailResponse, error) {
	// Validate BOM exists
	if _, err := s.repo.GetBomByID(ctx, bomID); err != nil {
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
		if err := s.repo.DeleteRoutingByItemID(ctx, item.ID); err != nil {
			return nil, err
		}
		if len(*req.ProcessRoutes) > 0 {
			rev, err := s.repo.GetLatestRevision(ctx, item.ID)
			if err != nil {
				return nil, err
			}
			var revID int64
			if rev != nil {
				revID = rev.ID
			}
			if err := s.createRouting(ctx, item.ID, revID, *req.ProcessRoutes); err != nil {
				return nil, err
			}
		}
	}

	// Upsert material spec when provided
	if req.MaterialSpec != nil {
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
			Level:      level,
			QPU:        line.QtyPerUniq,
			Asset:      s.buildAssetInfo(preload.assetByItemID(child.ID)),
			Status:     child.Status,
		}
		if rev, ok := preload.revisions[child.ID]; ok {
			row.Version = &rev.Revision
			if spec, ok := preload.specs[rev.ID]; ok {
				row.MaterialSpec = s.toSpecDetail(&spec)
			}
		}
		if routes, ok := preload.routes[child.ID]; ok {
			row.ProcessRoutes = routes
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
			OpSeq:         op.OpSeq,
			ProcessName:   processNames[op.ProcessID],
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
	revisionMap := make(map[int64]models.ItemRevision, len(revisions))
	revisionIDSet := make(map[int64]struct{}, len(revisions))
	for _, revision := range revisions {
		revisionMap[revision.ItemID] = revision
		revisionIDSet[revision.ID] = struct{}{}
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

	headers, err := s.repo.GetLatestRoutingHeadersByItemIDs(ctx, itemIDs)
	if err != nil {
		return nil, err
	}
	headerIDs := make([]int64, 0, len(headers))
	headerItemMap := make(map[int64]int64, len(headers))
	for _, header := range headers {
		headerIDs = append(headerIDs, header.ID)
		headerItemMap[header.ID] = header.ItemID
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

	routes := make(map[int64][]models.ProcessRouteDetail, len(headers))
	for _, header := range headers {
		ops := opByHeaderID[header.ID]
		if len(ops) == 0 {
			continue
		}
		mergedToolings := make([]models.RoutingOperationTooling, 0)
		for _, op := range ops {
			mergedToolings = append(mergedToolings, toolingsByOpID[op.ID]...)
		}
		routes[header.ItemID] = s.toRouteDetails(ops, mergedToolings, processNames, machineNames)
	}

	return &bomPreload{
		items:     itemMap,
		revisions: revisionMap,
		assets:    assetMap,
		specs:     specMap,
		routes:    routes,
		children:  children,
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

func uniqueInt64Keys(values map[int64]struct{}) []int64 {
	result := make([]int64, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
