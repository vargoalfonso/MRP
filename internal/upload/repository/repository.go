// Package repository provides data-access for the chunked upload module.
package repository

import (
	"context"

	"github.com/ganasa18/go-template/internal/upload/models"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IRepository interface {
	CreateSession(ctx context.Context, s *models.UploadSession) error
	GetSession(ctx context.Context, id uuid.UUID) (*models.UploadSession, error)
	UpdateSession(ctx context.Context, s *models.UploadSession) error

	SaveChunk(ctx context.Context, c *models.UploadChunk) error
	GetChunks(ctx context.Context, sessionID uuid.UUID) ([]models.UploadChunk, error)
	ChunkExists(ctx context.Context, sessionID uuid.UUID, index int) (bool, error)

	DeleteChunks(ctx context.Context, sessionID uuid.UUID) error
}

type repository struct{ db *gorm.DB }

func New(db *gorm.DB) IRepository { return &repository{db: db} }

func (r *repository) CreateSession(ctx context.Context, s *models.UploadSession) error {
	if err := r.db.WithContext(ctx).Create(s).Error; err != nil {
		return apperror.InternalWrap("CreateSession", err)
	}
	return nil
}

func (r *repository) GetSession(ctx context.Context, id uuid.UUID) (*models.UploadSession, error) {
	var s models.UploadSession
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&s).Error
	if err == gorm.ErrRecordNotFound {
		return nil, apperror.NotFound("upload session not found")
	}
	if err != nil {
		return nil, apperror.InternalWrap("GetSession", err)
	}
	return &s, nil
}

func (r *repository) UpdateSession(ctx context.Context, s *models.UploadSession) error {
	if err := r.db.WithContext(ctx).Save(s).Error; err != nil {
		return apperror.InternalWrap("UpdateSession", err)
	}
	return nil
}

func (r *repository) SaveChunk(ctx context.Context, c *models.UploadChunk) error {
	// Upsert: if same session+index arrives again (retry), overwrite
	if err := r.db.WithContext(ctx).
		Where(models.UploadChunk{SessionID: c.SessionID, ChunkIndex: c.ChunkIndex}).
		Assign(*c).
		FirstOrCreate(c).Error; err != nil {
		return apperror.InternalWrap("SaveChunk", err)
	}
	return nil
}

func (r *repository) GetChunks(ctx context.Context, sessionID uuid.UUID) ([]models.UploadChunk, error) {
	var chunks []models.UploadChunk
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Order("chunk_index ASC").
		Find(&chunks).Error; err != nil {
		return nil, apperror.InternalWrap("GetChunks", err)
	}
	return chunks, nil
}

func (r *repository) ChunkExists(ctx context.Context, sessionID uuid.UUID, index int) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.UploadChunk{}).
		Where("session_id = ? AND chunk_index = ?", sessionID, index).
		Count(&count).Error
	if err != nil {
		return false, apperror.InternalWrap("ChunkExists", err)
	}
	return count > 0, nil
}

func (r *repository) DeleteChunks(ctx context.Context, sessionID uuid.UUID) error {
	if err := r.db.WithContext(ctx).
		Where("session_id = ?", sessionID).
		Delete(&models.UploadChunk{}).Error; err != nil {
		return apperror.InternalWrap("DeleteChunks", err)
	}
	return nil
}
