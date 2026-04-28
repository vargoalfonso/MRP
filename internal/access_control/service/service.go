package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/ganasa18/go-template/internal/access_control/models"
	acmRepo "github.com/ganasa18/go-template/internal/access_control/repository"
	authModels "github.com/ganasa18/go-template/internal/auth/models"
	authRepo "github.com/ganasa18/go-template/internal/auth/repository"
	departmentRepo "github.com/ganasa18/go-template/internal/departement/repository"
	employeeRepo "github.com/ganasa18/go-template/internal/employee/repository"
	roleRepo "github.com/ganasa18/go-template/internal/role/repository"
	"github.com/google/uuid"
	"gorm.io/gorm"
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

	// ==============================
	// 🔥 1. VALIDASI ROLE
	// ==============================
	role, err := s.roleRepo.FindByID(ctx, *req.RoleID)
	if err != nil {
		return nil, fmt.Errorf("role not found")
	}

	// ==============================
	// 🔥 2. VALIDASI OPTIONAL DATA
	// ==============================
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

	// ==============================
	// 🔥 3. UPDATE ROLE KE USER (NEW LOGIC)
	// ==============================
	if req.EmployeeID != nil {

		user, err := s.authRepo.FindUserByEmployeeID(ctx, *req.EmployeeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {

				employeeID := strconv.FormatInt(*req.EmployeeID, 10)

				newUser := &authModels.User{
					UUID:       uuid.New().String(),
					Username:   employeeID,
					Email:      fmt.Sprintf("%s@local.user", employeeID),
					Roles:      "user",
					EmployeeID: &employeeID,
					Provider:   "local",
					IsVerified: true,
				}

				if err := s.authRepo.Create(ctx, newUser); err != nil {
					return nil, err
				}

				user = newUser

			} else {
				return nil, err
			}
		}

		// ❌ BELUM VERIFIED
		if !user.IsVerified {
			return nil, fmt.Errorf("user belum aktivasi akun (belum set password)")
		}

		// ✅ UPDATE ROLE
		err = s.authRepo.UpdateUserRole(ctx, user.ID, role.Name)
		if err != nil {
			return nil, err
		}
	}

	// ==============================
	// 🔥 4. CREATE ACM
	// ==============================
	data, err := s.repo.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateACMRequest) (*models.AccessControlMatrix, error) {

	// ==============================
	// 🔥 1. CEK ACM EXIST
	// ==============================
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("acm not found")
	}

	// ==============================
	// 🔥 2. VALIDASI ROLE
	// ==============================
	var roleName string
	if req.RoleID != nil {
		role, err := s.roleRepo.FindByID(ctx, *req.RoleID)
		if err != nil {
			return nil, fmt.Errorf("role not found")
		}
		roleName = role.Name
	}

	// ==============================
	// 🔥 3. VALIDASI EMPLOYEE
	// ==============================
	if req.EmployeeID != nil {
		_, err := s.employeeRepo.FindByID(ctx, *req.EmployeeID)
		if err != nil {
			return nil, fmt.Errorf("employee not found")
		}
	}

	// ==============================
	// 🔥 4. VALIDASI DEPARTMENT
	// ==============================
	if req.DepartmentID != nil {
		_, err := s.departmentRepo.FindByID(ctx, *req.DepartmentID)
		if err != nil {
			return nil, fmt.Errorf("department not found")
		}
	}

	// ==============================
	// 🔥 5. SYNC USER (IMPORTANT FIX)
	// ==============================
	targetEmployeeID := existing.EmployeeID
	if req.EmployeeID != nil {
		targetEmployeeID = req.EmployeeID
	}

	if targetEmployeeID != nil {

		user, err := s.authRepo.FindUserByEmployeeID(ctx, *targetEmployeeID)
		if err != nil {
			return nil, fmt.Errorf("user tidak ditemukan untuk employee ini")
		}

		// ❌ BELUM VERIFIED → TOLAK
		if !user.IsVerified {
			return nil, fmt.Errorf("user belum aktivasi akun")
		}

		// 🔥 UPDATE ROLE
		if req.RoleID != nil {
			err = s.authRepo.UpdateUserRole(ctx, user.ID, roleName)
			if err != nil {
				return nil, err
			}
		}
	}

	// ==============================
	// 🔥 6. UPDATE ACM
	// ==============================
	data, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	return data, nil
}

func (s *service) Delete(ctx context.Context, id int64) error {

	// ==============================
	// 🔥 1. CEK ACM EXIST
	// ==============================
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return fmt.Errorf("acm not found")
	}

	// ==============================
	// 🔥 2. AMBIL USER DARI EMPLOYEE
	// ==============================
	if existing.EmployeeID != nil {

		user, err := s.authRepo.FindUserByEmployeeID(ctx, *existing.EmployeeID)
		if err == nil && user != nil {

			// 🔥 DOWNGRADE ROLE → "user"
			err = s.authRepo.UpdateUserRole(ctx, user.ID, "user")
			if err != nil {
				return err
			}
		}
	}

	// ==============================
	// 🔥 3. DELETE ACM
	// ==============================
	err = s.repo.Delete(ctx, id)
	if err != nil {
		return err
	}

	return nil
}
