package service

import (
	"context"
	"sync"
	"time"

	"github.com/ganasa18/go-template/internal/role/models"
	roleRepo "github.com/ganasa18/go-template/internal/role/repository"
)

// permissionCacheTTL menentukan berapa lama hasil GetPermissions disimpan
// di memori. Middleware RequirePermission dipanggil pada SETIAP request,
// sehingga tanpa cache satu halaman yang memuat 3 endpoint sekaligus akan
// menembak tabel roles 3 kali dan cepat menghabiskan connection pool.
const permissionCacheTTL = 60 * time.Second

type permissionCacheEntry struct {
	permissions map[string]interface{}
	expiresAt   time.Time
}

// IRoleService defines all role operations
type IRoleService interface {
	GetAll(ctx context.Context) ([]models.Role, error)
	GetByID(ctx context.Context, id int64) (*models.Role, error)
	Create(ctx context.Context, req models.CreateRoleRequest) (*models.Role, error)
	Update(ctx context.Context, id int64, req models.UpdateRoleRequest) (*models.Role, error)
	Delete(ctx context.Context, id int64) error

	// 🔥 RBAC
	GetPermissions(ctx context.Context, role string) (map[string]interface{}, error)
	GetUsersByRoleID(ctx context.Context, roleID int64) ([]models.RoleUser, error)
}

// implementation
type service struct {
	repo roleRepo.IRoleRepository

	permMu    sync.RWMutex
	permCache map[string]permissionCacheEntry
}

// constructor (mirip auth.New)
func New(repo roleRepo.IRoleRepository) IRoleService {
	return &service{
		repo:      repo,
		permCache: make(map[string]permissionCacheEntry),
	}
}

// invalidatePermissionCache dipanggil setiap kali data role berubah supaya
// perubahan hak akses langsung berlaku dan tidak tertahan sampai TTL habis.
func (s *service) invalidatePermissionCache() {
	s.permMu.Lock()
	s.permCache = make(map[string]permissionCacheEntry)
	s.permMu.Unlock()
}

// =========================
// CRUD
// =========================

func (s *service) GetAll(ctx context.Context) ([]models.Role, error) {
	return s.repo.FindAll(ctx)
}

func (s *service) GetByID(ctx context.Context, id int64) (*models.Role, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *service) Create(ctx context.Context, req models.CreateRoleRequest) (*models.Role, error) {
	role, err := s.repo.Create(ctx, req)
	if err == nil {
		s.invalidatePermissionCache()
	}
	return role, err
}

func (s *service) Update(ctx context.Context, id int64, req models.UpdateRoleRequest) (*models.Role, error) {
	role, err := s.repo.Update(ctx, id, req)
	if err == nil {
		s.invalidatePermissionCache()
	}
	return role, err
}

func (s *service) Delete(ctx context.Context, id int64) error {
	err := s.repo.Delete(ctx, id)
	if err == nil {
		s.invalidatePermissionCache()
	}
	return err
}

// =========================
// RBAC (IMPORTANT)
// =========================

func (s *service) GetPermissions(ctx context.Context, role string) (map[string]interface{}, error) {
	// Ambil dari cache dulu.
	s.permMu.RLock()
	entry, found := s.permCache[role]
	s.permMu.RUnlock()

	if found && time.Now().Before(entry.expiresAt) {
		return entry.permissions, nil
	}

	permissions, err := s.repo.GetPermissionsByRole(ctx, role)
	if err != nil {
		return nil, err
	}

	s.permMu.Lock()
	s.permCache[role] = permissionCacheEntry{
		permissions: permissions,
		expiresAt:   time.Now().Add(permissionCacheTTL),
	}
	s.permMu.Unlock()

	return permissions, nil
}

func (s *service) GetUsersByRoleID(ctx context.Context, roleID int64) ([]models.RoleUser, error) {
	return s.repo.FindUsersByRoleID(ctx, roleID)
}
