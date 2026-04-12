// Package models defines domain structs for the chunked upload module.
package models

import (
	"time"

	"github.com/google/uuid"
)

// UploadSession represents one resumable upload attempt for a single file.
type UploadSession struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey"`
	ItemID      int64      `gorm:"not null"`
	AssetType   string     `gorm:"size:32;not null"` // drawing / photo / 3d-model / other
	FileName    string     `gorm:"size:255;not null"`
	MimeType    *string    `gorm:"size:100"`
	FileSize    int64      `gorm:"not null"` // bytes
	ChunkSize   int        `gorm:"not null"` // bytes per chunk
	TotalChunks int        `gorm:"not null"`
	Status      string     `gorm:"size:20;default:pending"` // pending/uploading/assembling/completed/failed/cancelled
	AssetID     *int64     `gorm:"index"`                   // if set, Complete replaces this asset (edit flow)
	FinalURL    *string    `gorm:"type:text"`
	ExpiresAt   time.Time  `gorm:"not null"`
	CreatedBy   *uuid.UUID `gorm:"type:uuid"`
	CreatedAt   time.Time
	UpdatedAt   time.Time

	Chunks []UploadChunk `gorm:"foreignKey:SessionID"`
}

func (UploadSession) TableName() string { return "upload_sessions" }

// UploadChunk is one received chunk within a session.
type UploadChunk struct {
	ID         int64     `gorm:"primaryKey;autoIncrement"`
	SessionID  uuid.UUID `gorm:"type:uuid;not null;index"`
	ChunkIndex int       `gorm:"not null"` // 0-based
	FilePath   string    `gorm:"type:text;not null"`
	Size       int64     `gorm:"not null"`
	Checksum   *string   `gorm:"size:64"` // optional MD5 hex from client
	UploadedAt time.Time `gorm:"not null"`
}

func (UploadChunk) TableName() string { return "upload_chunks" }

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

// CreateSessionRequest — body for POST /uploads/sessions
type CreateSessionRequest struct {
	ItemID      int64   `json:"item_id" validate:"required"`
	AssetType   string  `json:"asset_type" validate:"required,oneof=drawing photo 3d-model other"`
	FileName    string  `json:"file_name" validate:"required,max=255"`
	MimeType    *string `json:"mime_type"`
	FileSize    int64   `json:"file_size" validate:"required,min=1"`      // bytes
	ChunkSize   int     `json:"chunk_size" validate:"required,min=65536"` // min 64 KB
	TotalChunks int     `json:"total_chunks" validate:"required,min=1"`

	// AssetID is optional. When provided, Complete will update the existing asset
	// (file_url, asset_type) instead of inserting a new row — use this for edit/replace flow.
	AssetID *int64 `json:"asset_id"`
}

// CompleteSessionRequest — body for POST /uploads/sessions/:id/complete
type CompleteSessionRequest struct {
	// Optional: client can send the final checksum of the whole file
	FileChecksum *string `json:"file_checksum"`
}

// ---------------------------------------------------------------------------
// Response types
// ---------------------------------------------------------------------------

// SessionResponse is returned after creating or fetching a session.
type SessionResponse struct {
	SessionID     string   `json:"session_id"`
	ItemID        int64    `json:"item_id"`
	AssetType     string   `json:"asset_type"`
	FileName      string   `json:"file_name"`
	FileSize      int64    `json:"file_size"`
	ChunkSize     int      `json:"chunk_size"`
	TotalChunks   int      `json:"total_chunks"`
	UploadedChunks []int   `json:"uploaded_chunks"` // indices already uploaded (for resume)
	MissingChunks  []int   `json:"missing_chunks"`  // indices still needed
	Status        string   `json:"status"`
	FinalURL      *string  `json:"final_url"`
	ExpiresAt     string   `json:"expires_at"`
}

// ChunkResponse is returned after a chunk is accepted.
type ChunkResponse struct {
	SessionID  string `json:"session_id"`
	ChunkIndex int    `json:"chunk_index"`
	Received   int64  `json:"received_bytes"`
	TotalDone  int    `json:"total_chunks_done"`
	TotalNeeded int   `json:"total_chunks_needed"`
	Complete   bool   `json:"complete"` // true when all chunks are in
}
