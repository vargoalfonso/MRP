package service

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

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

	Approve(ctx context.Context, instanceID int64, userRoles []string) error
	Reject(ctx context.Context, instanceID int64, userRoles []string, note string) error
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

func (s *service) Approve(ctx context.Context, instanceID int64, userRoles []string) error {

	instance, err := s.repo.FindInstanceByID(ctx, instanceID)
	if err != nil {
		return err
	}

	progress := instance.ApprovalProgress

	// ==============================
	// 🔥 1. CARI CURRENT LEVEL YANG BELUM APPROVED
	// ==============================
	var currentIdx int = -1

	for i, lvl := range progress.Levels {

		// skip level kosong
		if lvl.Status == "skipped" {
			continue
		}

		if lvl.Status == "pending" {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 {
		return fmt.Errorf("semua level sudah diproses")
	}

	level := &progress.Levels[currentIdx]

	// ==============================
	// 🔥 2. VALIDASI ROLE USER
	// ==============================
	isAllowed := false
	for _, r := range userRoles {
		if r == level.Role {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("menunggu approval dari role '%s'", level.Role)
	}

	// ==============================
	// 🔥 3. APPROVE
	// ==============================
	level.Status = "approved"
	level.ApprovedAt = time.Now().Format(time.RFC3339)
	level.ApprovedBy = strings.Join(userRoles, ",")

	// ==============================
	// 🔥 4. CEK FINAL APPROVAL
	// ==============================
	isFinal := true

	for _, lvl := range progress.Levels {
		if lvl.Status == "pending" {
			isFinal = false
			break
		}
	}

	if isFinal {
		instance.Status = "approved"

		// 🔥 UPDATE ENTITY (DN)
		if err := s.repo.UpdateReferenceStatus(ctx, instance.ReferenceTable, instance.ReferenceID, "active"); err != nil {
			return err
		}

	} else {
		instance.CurrentLevel = currentIdx + 2 // next level
	}

	// ==============================
	// 🔥 5. SAVE
	// ==============================
	instance.ApprovalProgress = progress
	instance.UpdatedAt = time.Now()

	return s.repo.UpdateInstance(ctx, instance)
}

func (s *service) Reject(ctx context.Context, instanceID int64, userRoles []string, note string) error {

	instance, err := s.repo.FindInstanceByID(ctx, instanceID)
	if err != nil {
		return err
	}

	progress := instance.ApprovalProgress

	// 🔥 cari level aktif
	var currentIdx int = -1

	for i, lvl := range progress.Levels {
		if lvl.Status == "pending" {
			currentIdx = i
			break
		}
	}

	if currentIdx == -1 {
		return fmt.Errorf("tidak ada level yang bisa di reject")
	}

	level := &progress.Levels[currentIdx]

	// 🔥 cek role
	isAllowed := false
	for _, r := range userRoles {
		if r == level.Role {
			isAllowed = true
			break
		}
	}

	if !isAllowed {
		return fmt.Errorf("menunggu approval dari role '%s'", level.Role)
	}

	// 🔥 reject
	level.Status = "rejected"
	level.Note = note
	level.ApprovedAt = time.Now().Format(time.RFC3339)
	level.ApprovedBy = strings.Join(userRoles, ",")

	instance.Status = "rejected"

	// 🔥 update DN
	if err := s.repo.UpdateReferenceStatus(
		ctx,
		instance.ReferenceTable,
		instance.ReferenceID,
		"rejected",
	); err != nil {
		return err
	}

	instance.ApprovalProgress = progress
	instance.UpdatedAt = time.Now()

	return s.repo.UpdateInstance(ctx, instance)
}
