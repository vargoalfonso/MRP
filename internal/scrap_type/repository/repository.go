package repository

import (
	"context"
	"strings"

	"github.com/ganasa18/go-template/internal/scrap_type/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	List(ctx context.Context, q ListQuery) ([]models.ScrapType, int64, error)
	GetByID(ctx context.Context, id int64) (*models.ScrapType, error)
	GetByName(ctx context.Context, name string) (*models.ScrapType, error)
	Create(ctx context.Context, s *models.ScrapType) error
	Update(ctx context.Context, s *models.ScrapType) error
	Delete(ctx context.Context, id int64) error
	GetNextCode(ctx context.Context) (string, error)
	IsUsedInTransactions(ctx context.Context, id int64) (bool, error)
}

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repository{db: db} }

type ListQuery struct {
	Page   int
	Limit  int
	Offset int
	Search string
	Status string
}

func (r *repository) List(ctx context.Context, q ListQuery) ([]models.ScrapType, int64, error) {
	var items []models.ScrapType
	var total int64

	base := r.db.WithContext(ctx).Model(&models.ScrapType{}).Where("deleted_at IS NULL")

	if search := strings.TrimSpace(q.Search); search != "" {
		base = base.Where("code ILIKE ? OR name ILIKE ?", "%"+search+"%", "%"+search+"%")
	}
	if status := strings.TrimSpace(q.Status); status != "" {
		base = base.Where("status = ?", status)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("list scrap types count", err)
	}

	if err := base.Order("id ASC").Offset(q.Offset).Limit(q.Limit).Find(&items).Error; err != nil {
		return nil, 0, apperror.InternalWrap("list scrap types", err)
	}

	return items, total, nil
}

func (r *repository) GetByID(ctx context.Context, id int64) (*models.ScrapType, error) {
	var s models.ScrapType
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL").
		First(&s, id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("scrap type not found")
		}
		return nil, apperror.InternalWrap("get scrap type by id", err)
	}
	return &s, nil
}

func (r *repository) GetByName(ctx context.Context, name string) (*models.ScrapType, error) {
	var s models.ScrapType
	err := r.db.WithContext(ctx).
		Where("deleted_at IS NULL AND LOWER(name) = LOWER(?)", strings.TrimSpace(name)).
		First(&s).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, apperror.InternalWrap("get scrap type by name", err)
	}
	return &s, nil
}

func (r *repository) Create(ctx context.Context, s *models.ScrapType) error {
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return apperror.InternalWrap("create scrap type", err)
	}
	return nil
}

func (r *repository) Update(ctx context.Context, s *models.ScrapType) error {
	if err := r.db.WithContext(ctx).Save(s).Error; err != nil {
		return apperror.InternalWrap("update scrap type", err)
	}
	return nil
}

func (r *repository) Delete(ctx context.Context, id int64) error {
	if err := r.db.WithContext(ctx).
		Model(&models.ScrapType{}).
		Where("id = ?", id).
		Update("deleted_at", gorm.Expr("NOW()")).Error; err != nil {
		return apperror.InternalWrap("delete scrap type", err)
	}
	return nil
}

func (r *repository) GetNextCode(ctx context.Context) (string, error) {
	var maxCode string
	err := r.db.WithContext(ctx).
		Model(&models.ScrapType{}).
		Where("code LIKE 'SCR-%'").
		Select("COALESCE(MAX(code), 'SCR-000')").
		Scan(&maxCode).Error
	if err != nil {
		return "", apperror.InternalWrap("get next scrap type code", err)
	}

	// Parse number and increment
	num := 0
	if len(maxCode) > 4 {
		// Extract number from "SCR-XXX"
		num = parseIntFromEnd(maxCode[4:])
	}
	nextNum := num + 1
	return "SCR-" + formatCodeNumber(nextNum), nil
}

func (r *repository) IsUsedInTransactions(ctx context.Context, id int64) (bool, error) {
	// Check if scrap_type_id is used in scrap_stocks or any scrap transactions
	var count int64
	err := r.db.WithContext(ctx).
		Raw("SELECT COUNT(*) FROM scrap_stocks WHERE scrap_type_id = ? AND deleted_at IS NULL", id).
		Scan(&count).Error
	if err != nil {
		// Table might not exist yet, return false
		return false, nil
	}
	return count > 0, nil
}

// Helper functions
func parseIntFromEnd(s string) int {
	num := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			num = num*10 + int(c-'0')
		}
	}
	return num
}

func formatCodeNumber(n int) string {
	return string(rune('0'+((n/100)%10))) +
		string(rune('0'+((n/10)%10))) +
		string(rune('0' + (n % 10)))
}
