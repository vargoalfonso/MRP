package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/ganasa18/go-template/internal/approval_workflow/models"
	approvalWorkflowRepo "github.com/ganasa18/go-template/internal/approval_workflow/repository"
	roleRepo "github.com/ganasa18/go-template/internal/role/repository"
)

type IApprovalWorkflowService interface {
	GetAll(ctx context.Context) ([]models.ApprovalWorkflow, error)
	Create(ctx context.Context, req models.CreateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error)
	GetByID(ctx context.Context, id int64) (*models.ApprovalWorkflow, error)
	Update(ctx context.Context, id int64, req models.UpdateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error)
	Delete(ctx context.Context, id int64) error
}

// implementation
type service struct {
	repo     approvalWorkflowRepo.IApprovalWorkflowRepository
	roleRepo roleRepo.IRoleRepository
}

func New(repo approvalWorkflowRepo.IApprovalWorkflowRepository, roleRepo roleRepo.IRoleRepository) IApprovalWorkflowService {
	return &service{repo: repo, roleRepo: roleRepo}
}

// =========================
// CRUD
// =========================
func (s *service) GetAll(ctx context.Context) ([]models.ApprovalWorkflow, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.ApprovalWorkflow, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error) {

	// 🔥 CEK DUPLICATE
	_, err := s.repo.FindByActionName(ctx, req.ActionName)
	if err == nil {
		return nil, fmt.Errorf("workflow '%s' sudah ada", req.ActionName)
	}

	roles := []string{
		req.Level1Role,
		req.Level2Role,
		req.Level3Role,
		req.Level4Role,
	}

	// ✅ VALIDASI
	if err := validateLevel1Required(roles[0]); err != nil {
		return nil, err
	}

	if err := validateApprovalSequence(roles); err != nil {
		return nil, err
	}

	if err := s.validateRoles(ctx, roles); err != nil {
		return nil, err
	}

	// 🔥 SAVE
	return s.repo.Create(ctx, req)
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateApprovalWorkflowRequest) (*models.ApprovalWorkflow, error) {

	// ==============================
	// 🔥 1. GET EXISTING
	// ==============================
	existing, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("workflow not found")
	}

	// ==============================
	// 🔥 2. DETECT ROLE CHANGES
	// ==============================
	isChanged :=
		(req.Level1Role != nil && *req.Level1Role != existing.Level1Role) ||
			(req.Level2Role != nil && *req.Level2Role != existing.Level2Role) ||
			(req.Level3Role != nil && *req.Level3Role != existing.Level3Role) ||
			(req.Level4Role != nil && *req.Level4Role != existing.Level4Role)

	// ==============================
	// 🔥 3. VALIDASI ROLE (SEBELUM UPDATE 🔥)
	// ==============================
	if isChanged {
		roles := []string{}

		if req.Level1Role != nil {
			roles = append(roles, *req.Level1Role)
		}
		if req.Level2Role != nil {
			roles = append(roles, *req.Level2Role)
		}
		if req.Level3Role != nil {
			roles = append(roles, *req.Level3Role)
		}
		if req.Level4Role != nil {
			roles = append(roles, *req.Level4Role)
		}

		if err := validateLevel1Required(roles[0]); err != nil {
			return nil, err
		}
		// ==============================
		// 🔥 3. VALIDASI SEQUENCE (NO GAP)
		// ==============================
		if err := validateApprovalSequence(roles); err != nil {
			return nil, err
		}

		// ==============================
		// 🔥 4. VALIDASI ROLE EXIST
		// ==============================
		if err := s.validateRoles(ctx, roles); err != nil {
			return nil, err
		}

	}

	// ==============================
	// 🔥 4. UPDATE WORKFLOW (AMAN)
	// ==============================
	updated, err := s.repo.Update(ctx, id, req)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

func (s *service) Delete(ctx context.Context, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *service) validateRoles(ctx context.Context, roles []string) error {
	for _, role := range roles {
		if role == "" {
			continue // skip kosong
		}

		_, err := s.roleRepo.FindByName(ctx, role)
		if err != nil {
			return fmt.Errorf("role '%s' tidak ditemukan", role)
		}
	}
	return nil
}

func ToTableName(input string) string {

	// ==============================
	// 🔥 1. handle "Delivery Note"
	// ==============================
	s := strings.TrimSpace(input)
	s = strings.ReplaceAll(s, "-", " ")
	s = strings.ReplaceAll(s, "_", " ")

	// ==============================
	// 🔥 2. CamelCase → space
	// ==============================
	re := regexp.MustCompile(`([a-z0-9])([A-Z])`)
	s = re.ReplaceAllString(s, "${1} ${2}")

	// ==============================
	// 🔥 3. lower + join underscore
	// ==============================
	words := strings.Fields(s)
	snake := strings.ToLower(strings.Join(words, "_"))

	// ==============================
	// 🔥 4. plural simple
	// ==============================
	if strings.HasSuffix(snake, "y") {
		return strings.TrimSuffix(snake, "y") + "ies"
	}

	if strings.HasSuffix(snake, "s") {
		return snake
	}

	return snake + "s"
}

func getOrDefault(new *string, old string) string {
	if new != nil {
		return *new
	}
	return old
}

func validateApprovalSequence(roles []string) error {
	foundEmpty := false

	for i, role := range roles {

		if role == "" {
			foundEmpty = true
			continue
		}

		// ❌ kalau sudah pernah kosong lalu ada isi lagi
		if foundEmpty {
			return fmt.Errorf("level %d tidak boleh diisi setelah level kosong sebelumnya", i+1)
		}
	}

	return nil
}

func validateLevel1Required(role string) error {
	if strings.TrimSpace(role) == "" {
		return fmt.Errorf("level 1 role wajib diisi")
	}
	return nil
}
