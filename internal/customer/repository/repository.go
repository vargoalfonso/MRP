package repository

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/customer/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"gorm.io/gorm"
)

type IRepository interface {
	Create(ctx context.Context, customer *models.Customer) error
	FindByUUID(ctx context.Context, uuid string) (*models.Customer, error)
	List(ctx context.Context, filters models.CustomerListFilters) ([]models.Customer, int64, error)
	Update(ctx context.Context, customer *models.Customer) error
	Delete(ctx context.Context, customer *models.Customer) error
}

type repository struct {
	db *gorm.DB
}

func New(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) Create(ctx context.Context, customer *models.Customer) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("LOCK TABLE public.customers IN EXCLUSIVE MODE").Error; err != nil {
			return apperror.InternalWrap("lock customers table failed", err)
		}

		customerID, err := nextCustomerID(tx)
		if err != nil {
			return err
		}

		customer.CustomerID = customerID
		if err := tx.Create(customer).Error; err != nil {
			return apperror.InternalWrap("create customer failed", err)
		}

		return nil
	})
}

func (r *repository) FindByUUID(ctx context.Context, uuid string) (*models.Customer, error) {
	var customer models.Customer
	err := r.db.WithContext(ctx).Where("uuid = ?", uuid).First(&customer).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, apperror.NotFound("customer not found")
		}
		return nil, apperror.InternalWrap("find customer failed", err)
	}

	return &customer, nil
}

func (r *repository) List(ctx context.Context, filters models.CustomerListFilters) ([]models.Customer, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Customer{})

	if filters.Search != "" {
		search := "%" + strings.TrimSpace(filters.Search) + "%"
		query = query.Where(
			"customer_id ILIKE ? OR customer_name ILIKE ? OR phone_number ILIKE ?",
			search,
			search,
			search,
		)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, apperror.InternalWrap("count customers failed", err)
	}

	var customers []models.Customer
	err := query.Order("created_at DESC").Limit(filters.Limit).Offset(filters.Offset).Find(&customers).Error
	if err != nil {
		return nil, 0, apperror.InternalWrap("list customers failed", err)
	}

	return customers, total, nil
}

func (r *repository) Update(ctx context.Context, customer *models.Customer) error {
	if err := r.db.WithContext(ctx).Save(customer).Error; err != nil {
		return apperror.InternalWrap("update customer failed", err)
	}

	return nil
}

func (r *repository) Delete(ctx context.Context, customer *models.Customer) error {
	if err := r.db.WithContext(ctx).Delete(customer).Error; err != nil {
		return apperror.InternalWrap("delete customer failed", err)
	}

	return nil
}

func nextCustomerID(tx *gorm.DB) (string, error) {
	year := time.Now().Year()
	prefix := fmt.Sprintf("CUST-%d-", year)

	var latestRecord struct {
		CustomerID string
	}
	err := tx.Unscoped().Model(&models.Customer{}).
		Select("customer_id").
		Where("customer_id LIKE ?", prefix+"%").
		Order("customer_id DESC").
		Limit(1).
		Take(&latestRecord).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return "", apperror.InternalWrap("get latest customer id failed", err)
	}

	sequence := 1
	if err == nil && latestRecord.CustomerID != "" {
		parts := strings.Split(latestRecord.CustomerID, "-")
		if len(parts) == 3 {
			lastNumber, convErr := strconv.Atoi(parts[2])
			if convErr != nil {
				return "", apperror.InternalWrap("parse latest customer id failed", convErr)
			}
			sequence = lastNumber + 1
		}
	}

	return fmt.Sprintf("CUST-%d-%03d", year, sequence), nil
}
