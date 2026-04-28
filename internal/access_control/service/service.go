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

	employeeID, err := strconv.ParseInt(req.EmployeeID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("employee_id harus angka")
	}

	roleID, err := strconv.ParseInt(req.RoleID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("role_id harus angka")
	}

	departmentID, err := strconv.ParseInt(req.DepartmentID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("department_id harus angka")
	}

	// ==============================
	// 🔥 1. VALIDASI ROLE
	// ==============================
	role, err := s.roleRepo.FindByID(ctx, roleID)
	if err != nil {
		return nil, fmt.Errorf("role not found")
	}

	// ==============================
	// 🔥 2. VALIDASI OPTIONAL DATA
	// ==============================
	if req.EmployeeID != "" {
		_, err := s.employeeRepo.FindByID(ctx, employeeID)
		if err != nil {
			return nil, fmt.Errorf("employee not found")
		}
	}

	if req.DepartmentID != "" {
		_, err := s.departmentRepo.FindByID(ctx, departmentID)
		if err != nil {
			return nil, fmt.Errorf("department not found")
		}
	}

	// ==============================
	// 🔥 3. UPDATE ROLE KE USER (NEW LOGIC)
	// ==============================
	if req.EmployeeID != "" {

		user, err := s.authRepo.FindUserByEmployeeID(ctx, employeeID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {

				employeeID := strconv.FormatInt(employeeID, 10)

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

func (s *service) Update(
	ctx context.Context,
	id int64,
	req models.UpdateACMRequest,
) (*models.AccessControlMatrix, error) {

	// ==============================
	// 🔥 1. CEK ACM EXIST
	// ==============================
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("acm not found")
	}

	var (
		employeeID   int64
		roleID       int64
		departmentID int64
		roleName     string
	)

	// ==============================
	// 🔥 2. PARSE REQUEST STRING -> INT
	// ==============================
	if req.EmployeeID != "" {
		employeeID, err = strconv.ParseInt(req.EmployeeID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("employee_id harus angka")
		}
	}

	if req.RoleID != "" {
		roleID, err = strconv.ParseInt(req.RoleID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("role_id harus angka")
		}
	}

	if req.DepartmentID != "" {
		departmentID, err = strconv.ParseInt(req.DepartmentID, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("department_id harus angka")
		}
	}

	// ==============================
	// 🔥 3. VALIDASI ROLE
	// ==============================
	if req.RoleID != "" {
		role, err := s.roleRepo.FindByID(ctx, roleID)
		if err != nil {
			return nil, fmt.Errorf("role not found")
		}

		roleName = role.Name
	}

	// ==============================
	// 🔥 4. VALIDASI EMPLOYEE
	// ==============================
	if req.EmployeeID != "" {
		_, err := s.employeeRepo.FindByID(ctx, employeeID)
		if err != nil {
			return nil, fmt.Errorf("employee not found")
		}
	}

	// ==============================
	// 🔥 5. VALIDASI DEPARTMENT
	// ==============================
	if req.DepartmentID != "" {
		_, err := s.departmentRepo.FindByID(ctx, departmentID)
		if err != nil {
			return nil, fmt.Errorf("department not found")
		}
	}

	// ==============================
	// 🔥 6. SYNC USER
	// ==============================
	targetEmployeeID := existing.EmployeeID

	if req.EmployeeID != "" {
		targetEmployeeID = employeeID
	}

	if req.EmployeeID != "" {
		targetEmployeeID = employeeID
	}

	user, err := s.authRepo.FindUserByEmployeeID(ctx, targetEmployeeID)
	if err != nil {
		return nil, fmt.Errorf("user tidak ditemukan untuk employee ini")
	}

	if !user.IsVerified {
		return nil, fmt.Errorf("user belum aktivasi akun")
	}

	if req.RoleID != "" {
		err = s.authRepo.UpdateUserRole(ctx, user.ID, roleName)
		if err != nil {
			return nil, err
		}
	}

	// ==============================
	// 🔥 7. UPDATE ACM
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
	user, err := s.authRepo.FindUserByEmployeeID(ctx, existing.EmployeeID)
	if err == nil && user != nil {

		// 🔥 DOWNGRADE ROLE → "user"
		if err := s.authRepo.UpdateUserRole(ctx, user.ID, "user"); err != nil {
			return err
		}
	}

	// ==============================
	// 🔥 3. DELETE ACM
	// ==============================
	if err := s.repo.Delete(ctx, id); err != nil {
		return err
	}

	return nil
}
