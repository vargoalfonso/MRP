package service

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	authModels "github.com/ganasa18/go-template/internal/auth/models"
	authRepo "github.com/ganasa18/go-template/internal/auth/repository"
	"github.com/ganasa18/go-template/internal/employee/models"
	employeeRepo "github.com/ganasa18/go-template/internal/employee/repository"
	email "github.com/ganasa18/go-template/pkg/email"
	"github.com/google/uuid"
)

type IEmployeeService interface {
	GetAll(ctx context.Context) ([]models.Employee, error)
	GetByID(ctx context.Context, id int64) (*models.Employee, error)
	Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.Employee, error)
	Update(ctx context.Context, id int64, req models.UpdateEmployeeRequest) (*models.Employee, error)
	Delete(ctx context.Context, id int64) error
}

type service struct {
	repo     employeeRepo.IEmployeeRepository
	authRepo authRepo.IRepository
}

func New(repo employeeRepo.IEmployeeRepository, authRepo authRepo.IRepository) IEmployeeService {
	return &service{
		repo:     repo,
		authRepo: authRepo,
	}
}

//
// =========================
// CRUD
// =========================
//

func (s *service) GetAll(ctx context.Context) ([]models.Employee, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.Employee, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateEmployeeRequest) (*models.Employee, error) {
	return s.repo.Update(ctx, id, req)
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

//
// =========================
// 🔥 CREATE EMPLOYEE FLOW
// =========================
//

func (s *service) Create(ctx context.Context, req models.CreateEmployeeRequest) (*models.Employee, error) {
	var employee *models.Employee

	err := s.repo.Tx(ctx, func(txRepo employeeRepo.IEmployeeRepository) error {

		// ==============================
		// 🔥 1. VALIDASI EMAIL
		// ==============================
		exist, err := txRepo.IsEmailExist(ctx, req.Email)
		if err != nil {
			return err
		}
		if exist {
			return fmt.Errorf("email sudah terdaftar")
		}

		// ==============================
		// 🔥 2. CREATE EMPLOYEE
		// ==============================
		emp, err := txRepo.Create(ctx, req)
		if err != nil {
			return err
		}
		employee = emp

		// ==============================
		// 🔥 3. CREATE USER (DALAM TX)
		// ==============================
		user := authModels.User{
			Username:   req.Email,
			Email:      req.Email,
			Password:   "",
			Roles:      "user",
			IsVerified: false,
		}

		// 🔥 IMPORTANT: harus pakai TX
		if err := s.authRepo.Create(ctx, &user); err != nil {
			return err
		}

		// ==============================
		// 🔥 4. GENERATE TOKEN
		// ==============================
		token := uuid.NewString()

		activation := models.UserActivation{
			UserID:    user.ID,
			Token:     token,
			ExpiredAt: time.Now().Add(24 * time.Hour),
			Used:      false,
			CreatedAt: time.Now(),
		}

		if err := txRepo.SaveActivationToken(ctx, &activation); err != nil {
			return err
		}

		// ==============================
		// 🔥 5. SEND EMAIL (ASYNC)
		// ==============================
		go func(to, name, token string) {
			link := fmt.Sprintf("%s/set-password?token=%s", os.Getenv("BASE_URL"), token)

			body := fmt.Sprintf(`
				<h2>Welcome %s 👋</h2>
				<p>Akun Anda sudah dibuat.</p>
				<p>Klik tombol di bawah untuk set password:</p>

				<a href="%s" style="padding:10px 20px;background:#28a745;color:white;text-decoration:none;border-radius:6px;">
					Set Password
				</a>

				<p>Link berlaku 24 jam</p>
			`, name, link)

			if err := email.SendEmail(to, "Set Password Account", body); err != nil {
				log.Println("failed send email:", err)
			}
		}(req.Email, req.FullName, token)

		return nil
	})

	if err != nil {
		return nil, err
	}

	return employee, nil
}
