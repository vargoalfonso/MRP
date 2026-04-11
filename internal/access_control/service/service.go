package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ganasa18/go-template/internal/access_control/models"
	acmRepo "github.com/ganasa18/go-template/internal/access_control/repository"
	authModels "github.com/ganasa18/go-template/internal/auth/models"
	authRepo "github.com/ganasa18/go-template/internal/auth/repository"
	departmentRepo "github.com/ganasa18/go-template/internal/departement/repository"
	employeeRepo "github.com/ganasa18/go-template/internal/employee/repository"
	roleRepo "github.com/ganasa18/go-template/internal/role/repository"
	"golang.org/x/crypto/bcrypt"
)

type IACMService interface {
	GetAll(ctx context.Context) ([]models.AccessControlMatrix, error)
	GetByID(ctx context.Context, id int64) (*models.AccessControlMatrix, error)
	Create(ctx context.Context, req models.CreateACMRequest) (*models.AccessControlMatrix, error)
	Update(ctx context.Context, id int64, req models.UpdateACMRequest) (*models.AccessControlMatrix, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo           acmRepo.IACMRepository
	roleRepo       roleRepo.IRoleRepository
	authRepo       authRepo.IRepository
	employeeRepo   employeeRepo.IEmployeeRepository
	departmentRepo departmentRepo.IDepartementRepository
}

func New(repo acmRepo.IACMRepository, roleRepo roleRepo.IRoleRepository, authRepo authRepo.IRepository, employeeRepo employeeRepo.IEmployeeRepository, departmentRepo departmentRepo.IDepartementRepository) IACMService {
	return &service{repo: repo, roleRepo: roleRepo, authRepo: authRepo, employeeRepo: employeeRepo, departmentRepo: departmentRepo}
}

// =========================
// CRUD
// =========================
func (s *service) GetAll(ctx context.Context) ([]models.AccessControlMatrix, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.AccessControlMatrix, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateACMRequest) (*models.AccessControlMatrix, error) {
	// 🔥 1. ambil role (buat dapet name)
	role, err := s.roleRepo.FindByID(ctx, *req.RoleID)
	if err != nil {
		return nil, fmt.Errorf("role not found")
	}

	if req.EmployeeID != nil {
		_, err := s.employeeRepo.FindByID(ctx, *req.EmployeeID)
		if err != nil {
			return nil, fmt.Errorf("employee not found")
		}
	}

	if req.DepartmentID != nil {
		_, err := s.departmentRepo.FindByID(ctx, *req.DepartmentID)
		if err != nil {
			return nil, fmt.Errorf("department not found")
		}
	}

	// 🔥 2. generate user data
	username := generateUsername(req.FullName)
	email := username + "@gmail.com"
	password := "password123"

	// 🔐 hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 🔥 3. create user
	user := authModels.User{
		Username:   username,
		Email:      email,
		Password:   string(hashedPassword),
		Roles:      role.Name,
		EmployeeID: nil,
	}

	// kalau employee_id ada → assign
	if req.EmployeeID != nil {
		empIDStr := strconv.FormatInt(*req.EmployeeID, 10)
		user.EmployeeID = &empIDStr
	}

	err = s.authRepo.Create(ctx, &user)
	if err != nil {
		return nil, err
	}

	// 🔥 4. create ACM
	data, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateACMRequest) (*models.AccessControlMatrix, error) {

	// 🔥 1. cek ACM exist
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("acm not found")
	}

	// 🔥 2. validasi role
	if req.RoleID != nil {
		_, err := s.roleRepo.FindByID(ctx, *req.RoleID)
		if err != nil {
			return nil, fmt.Errorf("role not found")
		}
	}

	// 🔥 3. validasi employee
	if req.EmployeeID != nil {
		_, err := s.employeeRepo.FindByID(ctx, *req.EmployeeID)
		if err != nil {
			return nil, fmt.Errorf("employee not found")
		}
	}

	// 🔥 4. validasi department
	if req.DepartmentID != nil {
		_, err := s.departmentRepo.FindByID(ctx, *req.DepartmentID)
		if err != nil {
			return nil, fmt.Errorf("department not found")
		}
	}

	// 🔥 6. update USER (optional sync)
	if req.FullName != "" || req.EmployeeID != nil || req.RoleID != nil {

		// 🔥 ambil data lama (untuk cari user)
		baseUsername := generateUsername(existing.FullName)
		baseEmail := baseUsername + "@gmail.com"

		user, err := s.authRepo.FindByEmail(ctx, baseEmail)
		if err != nil {
			return nil, fmt.Errorf("associated user not found")
		}

		// 🔥 update username + email (kalau fullname berubah)
		if req.FullName != "" {
			newUsername := generateUsername(req.FullName)

			// ⚠️ handle duplicate username
			existingUser, _ := s.authRepo.FindByEmail(ctx, baseEmail)
			if existingUser != nil && existingUser.ID != user.ID {
				newUsername = fmt.Sprintf("%s%d", newUsername, time.Now().Unix())
			}

			user.Username = newUsername
			user.Email = newUsername + "@gmail.com"
		}

		// 🔥 update employee_id
		if req.EmployeeID != nil {
			empIDStr := strconv.FormatInt(*req.EmployeeID, 10)
			user.EmployeeID = &empIDStr
		}

		// 🔥 update role
		if req.RoleID != nil {
			role, err := s.roleRepo.FindByID(ctx, *req.RoleID)
			if err != nil {
				return nil, fmt.Errorf("role not found")
			}
			user.Roles = role.Name
		}

		// 🔥 save
		err = s.authRepo.Update(ctx, user.ID, user)
		if err != nil {
			return nil, err
		}
	}

	// 🔥 5. update ACM
	data, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *service) Delete(ctx context.Context, id int64) error {

	// 🔥 1. cek ACM exist
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("acm not found")
	}

	// 🔥 2. generate email dari fullname
	username := generateUsername(existing.FullName)
	email := username + "@gmail.com"

	// 🔥 3. cari user
	user, err := s.authRepo.FindByEmail(ctx, email)
	fmt.Printf("DEBUG: Looking for user with email %s, found: %v, error: %v\n", email, user, err)
	if err == nil && user != nil {

		// 🔥 4. delete user
		err = s.authRepo.Delete(ctx, user.Email)
		if err != nil {
			return err
		}
	}

	// 🔥 5. delete ACM
	err = s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	return nil
}

func generateUsername(fullName string) string {
	// hapus spasi + lowercase
	return strings.ToLower(strings.ReplaceAll(fullName, " ", ""))
}
