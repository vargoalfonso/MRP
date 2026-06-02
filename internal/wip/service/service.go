package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/wip/models"
	wipRepo "github.com/ganasa18/go-template/internal/wip/repository"
	"github.com/google/uuid"
)

type IWIPService interface {
	// WIP
	GetAll(ctx context.Context, page, limit int) ([]models.WIPListResponse, int64, error)
	GetByID(ctx context.Context, id int64) (*models.WIPDetailResponse, error)
	Create(ctx context.Context, req models.CreateWIPRequest) (*models.WIP, error)
	Update(ctx context.Context, id int64, req models.UpdateWIPRequest) (*models.WIP, error)
	Delete(ctx context.Context, id int64) error

	// ITEMS
	GetItems(ctx context.Context, wipID int64) ([]models.WIPItem, error)

	// SCAN
	Scan(ctx context.Context, req models.ScanRequest) error
}

// implementation
type service struct {
	repo wipRepo.IWIPRepository
}

func New(repo wipRepo.IWIPRepository) IWIPService {
	return &service{repo: repo}
}

func (s *service) GetAll(ctx context.Context, page, limit int) ([]models.WIPListResponse, int64, error) {
	return s.repo.FindAllWIPPaginated(ctx, page, limit)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.WIPDetailResponse, error) {
	if id == 0 {
		return nil, fmt.Errorf("invalid id")
	}
	return s.repo.FindWIPByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateWIPRequest) (*models.WIP, error) {

	if req.WoID == 0 {
		return nil, fmt.Errorf("wo_id required")
	}

	tx := s.repo.BeginTx(ctx)
	defer tx.Rollback()

	wip, err := tx.CreateWIP(ctx, models.CreateWIPRequest{
		WoID: req.WoID,
	})
	if err != nil {
		return nil, err
	}

	for _, item := range req.Items {

		if len(item.ProcessFlow) == 0 {
			continue
		}

		// sort by op_seq (biar konsisten)
		sort.Slice(item.ProcessFlow, func(i, j int) bool {
			return item.ProcessFlow[i].OpSeq < item.ProcessFlow[j].OpSeq
		})

		seq := 1 // 🔥 reset per item

		for _, p := range item.ProcessFlow {

			wipItem := models.WIPItem{
				WipID: wip.ID,

				Uniq: item.Uniq,

				PackingNumber: item.KanbanNumber,
				WipType:       item.WipType,

				ProcessName: p.ProcessName,
				MachineName: p.MachineName,
				OpSeq:       p.OpSeq,

				Seq: seq,

				UOM: item.UOM,

				Stock: item.Stock,

				QtyRemaining: item.Stock,
				Status:       "queue",
			}

			_, err := tx.InsertWIPItem(ctx, wipItem)
			if err != nil {
				return nil, err
			}

			seq++ // 🔥 increment
		}
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}

	return wip, nil
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateWIPRequest) (*models.WIP, error) {

	if id == 0 {
		return nil, fmt.Errorf("invalid id")
	}

	return s.repo.UpdateWIP(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {

	if id == 0 {
		return fmt.Errorf("invalid id")
	}

	return s.repo.DeleteWIP(ctx, id)
}

func (s *service) GetItems(ctx context.Context, wipID int64) ([]models.WIPItem, error) {

	if wipID == 0 {
		return nil, fmt.Errorf("wip_id required")
	}

	return s.repo.FindItemsByWIP(ctx, wipID)
}

func (s *service) Scan(ctx context.Context, req models.ScanRequest) error {

	if req.WipItemID == 0 {
		return fmt.Errorf("wip_item_id required")
	}

	if req.Qty <= 0 {
		return fmt.Errorf("qty must be > 0")
	}

	item, err := s.repo.FindItemByID(ctx, req.WipItemID)
	if err != nil {
		return err
	}

	switch req.Action {

	case "scan_in":
		item.QtyIn += req.Qty
		item.QtyRemaining += req.Qty
		item.Status = "process"

	case "scan_out":

		if item.QtyRemaining < req.Qty {
			return fmt.Errorf("insufficient qty")
		}

		item.QtyOut += req.Qty
		item.QtyRemaining -= req.Qty

		if item.QtyRemaining == 0 {

			// default selesai WIP
			item.Status = "done"

			// hanya parent assembly masuk FG
			if isAssemblyProcess(item.ProcessName) {

				itemInfo, err := s.repo.GetItemInfo(ctx, item.Uniq)
				if err != nil {
					return err
				}

				isParent, err := s.repo.IsParentItem(ctx, itemInfo.ID)
				if err != nil {
					return err
				}

				// hanya parent masuk FG
				if isParent {

					WIP, _ := s.repo.GetWOIDByWIPID(ctx, item.WipID)

					wo, _ := s.repo.GetWO(ctx, WIP.ID)

					now := time.Now()

					fg := models.FinishedGoods{
						UUID:       uuid.New().String(),
						UniqCode:   item.Uniq,
						ItemID:     item.ID,
						PartNumber: itemInfo.PartNumber,
						PartName:   itemInfo.PartName,
						Model:      itemInfo.Model,
						WONumber:   wo,
						StockQty:   float64(req.Qty),
						UOM:        item.UOM,
						Status:     "AVAILABLE",
						CreatedAt:  now,
						UpdatedAt:  now,
					}

					err = s.repo.InsertFinishedGoods(ctx, fg)
					if err != nil {
						return err
					}

					item.Status = "fg"
				}
			}
		}

	default:
		return fmt.Errorf("invalid action")
	}

	err = s.repo.UpdateItemScan(ctx, item.ID, models.UpdateWIPItemScan{
		QtyIn:        item.QtyIn,
		QtyOut:       item.QtyOut,
		QtyRemaining: item.QtyRemaining,
		Status:       item.Status,
	})
	if err != nil {
		return err
	}

	_, err = s.repo.CreateLog(ctx, models.CreateWIPLogRequest{
		WipItemID: item.ID,
		Action:    req.Action,
		Qty:       req.Qty,
	})

	return err
}

func isAssemblyProcess(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))

	return name == "assembly" ||
		name == "assy" ||
		name == "final assembly"
}
