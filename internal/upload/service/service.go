// Package service implements business logic for resumable chunked file uploads.
//
// Upload flow:
//  1. POST /uploads/sessions                    → CreateSession  → returns session_id + missing_chunks
//  2. GET  /uploads/sessions/:id                → GetSession     → returns progress (for resume)
//  3. POST /uploads/sessions/:id/chunks/:index  → UploadChunk   → streams chunk to disk
//  4. POST /uploads/sessions/:id/complete       → Complete       → assembles file → creates item_asset
//  5. DELETE /uploads/sessions/:id              → Cancel         → removes tmp files
package service

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	bomModels "github.com/ganasa18/go-template/internal/billmaterial/models"
	bomRepo "github.com/ganasa18/go-template/internal/billmaterial/repository"
	"github.com/ganasa18/go-template/internal/upload/models"
	"github.com/ganasa18/go-template/internal/upload/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/google/uuid"
)

// SessionTTL is how long an incomplete session lives before it expires.
const SessionTTL = 24 * time.Hour

// ChunkDir is the base directory for temporary chunk files.
// Override via env/config if needed.
const ChunkDir = "tmp/uploads"

// FinalDir is where assembled files are stored and served from.
const FinalDir = "uploads/assets"

type IService interface {
	CreateSession(ctx context.Context, req models.CreateSessionRequest, createdBy *uuid.UUID) (*models.SessionResponse, error)
	GetSession(ctx context.Context, sessionID uuid.UUID) (*models.SessionResponse, error)
	UploadChunk(ctx context.Context, sessionID uuid.UUID, index int, body io.Reader, clientChecksum string) (*models.ChunkResponse, error)
	Complete(ctx context.Context, sessionID uuid.UUID, req models.CompleteSessionRequest) (*models.SessionResponse, error)
	Cancel(ctx context.Context, sessionID uuid.UUID) error
}

type service struct {
	repo          repository.IRepository
	bomRepo       bomRepo.IRepository
	maxChunkBytes int64
}

func New(repo repository.IRepository, bomRepository bomRepo.IRepository, maxChunkBytes int64) IService {
	return &service{repo: repo, bomRepo: bomRepository, maxChunkBytes: maxChunkBytes}
}

// ---------------------------------------------------------------------------
// CreateSession — Step 1
// ---------------------------------------------------------------------------

func (s *service) CreateSession(ctx context.Context, req models.CreateSessionRequest, createdBy *uuid.UUID) (*models.SessionResponse, error) {
	// Validate item exists
	if _, err := s.bomRepo.GetItemByID(ctx, req.ItemID); err != nil {
		return nil, err
	}

	sess := &models.UploadSession{
		ID:          uuid.New(),
		ItemID:      req.ItemID,
		AssetType:   req.AssetType,
		FileName:    req.FileName,
		MimeType:    req.MimeType,
		FileSize:    req.FileSize,
		ChunkSize:   req.ChunkSize,
		TotalChunks: req.TotalChunks,
		Status:      "pending",
		AssetID:     req.AssetID,
		ExpiresAt:   time.Now().Add(SessionTTL),
		CreatedBy:   createdBy,
	}
	if err := s.repo.CreateSession(ctx, sess); err != nil {
		return nil, err
	}

	// Ensure tmp directory exists
	tmpDir := filepath.Join(ChunkDir, sess.ID.String())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, apperror.InternalWrap("mkdir tmp", err)
	}

	return s.toResponse(sess, nil), nil
}

// ---------------------------------------------------------------------------
// GetSession — for resume: tells client which chunks are still missing
// ---------------------------------------------------------------------------

func (s *service) GetSession(ctx context.Context, sessionID uuid.UUID) (*models.SessionResponse, error) {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.checkExpiry(sess); err != nil {
		return nil, err
	}

	chunks, err := s.repo.GetChunks(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	return s.toResponse(sess, chunks), nil
}

// ---------------------------------------------------------------------------
// UploadChunk — Step 3 (called once per chunk, may be retried)
// ---------------------------------------------------------------------------

func (s *service) UploadChunk(ctx context.Context, sessionID uuid.UUID, index int, body io.Reader, clientChecksum string) (*models.ChunkResponse, error) {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.checkExpiry(sess); err != nil {
		return nil, err
	}
	if sess.Status == "completed" || sess.Status == "cancelled" {
		return nil, apperror.BadRequest("session is already " + sess.Status)
	}
	if index < 0 || index >= sess.TotalChunks {
		return nil, apperror.BadRequest(fmt.Sprintf("chunk_index %d out of range [0, %d)", index, sess.TotalChunks))
	}

	// Write chunk to disk
	tmpDir := filepath.Join(ChunkDir, sessionID.String())
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return nil, apperror.InternalWrap("mkdir tmp", err)
	}
	chunkPath := filepath.Join(tmpDir, fmt.Sprintf("%05d.chunk", index))

	f, err := os.Create(chunkPath)
	if err != nil {
		return nil, apperror.InternalWrap("create chunk file", err)
	}
	defer f.Close()

	limited := io.LimitReader(body, s.maxChunkBytes)

	// Tee through MD5 hasher while writing
	hasher := md5.New()
	written, err := io.Copy(io.MultiWriter(f, hasher), limited)
	if err != nil {
		return nil, apperror.InternalWrap("write chunk", err)
	}
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))

	// Optional: verify checksum if client sent one
	if clientChecksum != "" && clientChecksum != actualChecksum {
		_ = os.Remove(chunkPath)
		return nil, apperror.BadRequest(fmt.Sprintf("checksum mismatch for chunk %d: got %s want %s", index, actualChecksum, clientChecksum))
	}

	chunk := &models.UploadChunk{
		SessionID:  sessionID,
		ChunkIndex: index,
		FilePath:   chunkPath,
		Size:       written,
		Checksum:   &actualChecksum,
		UploadedAt: time.Now(),
	}
	if err := s.repo.SaveChunk(ctx, chunk); err != nil {
		return nil, err
	}

	// Update session status to uploading on first chunk
	if sess.Status == "pending" {
		sess.Status = "uploading"
		_ = s.repo.UpdateSession(ctx, sess)
	}

	// Count how many chunks we have now
	chunks, _ := s.repo.GetChunks(ctx, sessionID)
	done := len(chunks)
	complete := done >= sess.TotalChunks

	return &models.ChunkResponse{
		SessionID:   sessionID.String(),
		ChunkIndex:  index,
		Received:    written,
		TotalDone:   done,
		TotalNeeded: sess.TotalChunks,
		Complete:    complete,
	}, nil
}

// ---------------------------------------------------------------------------
// Complete — Step 4: assemble chunks → save final file → create item_asset
// ---------------------------------------------------------------------------

func (s *service) Complete(ctx context.Context, sessionID uuid.UUID, _ models.CompleteSessionRequest) (*models.SessionResponse, error) {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if err := s.checkExpiry(sess); err != nil {
		return nil, err
	}
	if sess.Status == "completed" {
		chunks, chunkErr := s.repo.GetChunks(ctx, sessionID)
		if chunkErr != nil {
			return nil, chunkErr
		}
		return s.toResponse(sess, chunks), nil
	}
	if sess.Status == "cancelled" {
		return nil, apperror.BadRequest("session was cancelled")
	}

	// Verify all chunks are present
	chunks, err := s.repo.GetChunks(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if len(chunks) != sess.TotalChunks {
		missing := s.missingIndices(chunks, sess.TotalChunks)
		return nil, apperror.BadRequest(fmt.Sprintf("missing chunks: %v", missing))
	}

	// Mark as assembling
	sess.Status = "assembling"
	_ = s.repo.UpdateSession(ctx, sess)

	// Assemble file
	finalURL, err := s.assembleChunks(sess, chunks)
	if err != nil {
		sess.Status = "failed"
		_ = s.repo.UpdateSession(ctx, sess)
		return nil, err
	}

	// Create or update item_asset record
	if sess.AssetID != nil {
		// Edit flow: replace file_url on the existing asset
		if err := s.bomRepo.UpdateAsset(ctx, *sess.AssetID, finalURL, sess.AssetType); err != nil {
			slog.Error("upload complete: failed to update item_asset", slog.Int64("asset_id", *sess.AssetID), slog.Any("error", err))
		}
	} else {
		// Create flow: insert new asset row
		asset := &bomModels.ItemAsset{
			ItemID:    sess.ItemID,
			AssetType: sess.AssetType,
			FileURL:   finalURL,
			Status:    "Active",
		}
		if err := s.bomRepo.CreateAsset(ctx, asset); err != nil {
			slog.Error("upload complete: failed to create item_asset", slog.Any("error", err))
		}
	}

	// Update session
	sess.Status = "completed"
	sess.FinalURL = &finalURL
	_ = s.repo.UpdateSession(ctx, sess)

	// Cleanup tmp chunks
	s.cleanupTmp(sess.ID)

	return s.toResponse(sess, chunks), nil
}

// ---------------------------------------------------------------------------
// Cancel — remove tmp files and mark session cancelled
// ---------------------------------------------------------------------------

func (s *service) Cancel(ctx context.Context, sessionID uuid.UUID) error {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return err
	}
	if sess.Status == "completed" {
		return apperror.BadRequest("cannot cancel a completed session")
	}

	sess.Status = "cancelled"
	_ = s.repo.UpdateSession(ctx, sess)
	s.cleanupTmp(sessionID)
	return nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (s *service) assembleChunks(sess *models.UploadSession, chunks []models.UploadChunk) (string, error) {
	// Final path: uploads/assets/{item_id}/{session_id}_{filename}
	dir := filepath.Join(FinalDir, fmt.Sprintf("%d", sess.ItemID))
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", apperror.InternalWrap("mkdir final", err)
	}
	finalPath := filepath.Join(dir, fmt.Sprintf("%s_%s", sess.ID.String()[:8], sess.FileName))

	out, err := os.Create(finalPath)
	if err != nil {
		return "", apperror.InternalWrap("create final file", err)
	}
	defer out.Close()

	for _, chunk := range chunks {
		f, err := os.Open(chunk.FilePath)
		if err != nil {
			return "", apperror.InternalWrap(fmt.Sprintf("open chunk %d", chunk.ChunkIndex), err)
		}
		if _, err := io.Copy(out, f); err != nil {
			f.Close()
			return "", apperror.InternalWrap(fmt.Sprintf("copy chunk %d", chunk.ChunkIndex), err)
		}
		f.Close()
	}

	// Return a relative URL (prepend your static file server prefix in real deployments)
	return "/" + filepath.ToSlash(finalPath), nil
}

func (s *service) cleanupTmp(sessionID uuid.UUID) {
	tmpDir := filepath.Join(ChunkDir, sessionID.String())
	if err := os.RemoveAll(tmpDir); err != nil {
		slog.Warn("upload cleanup: failed to remove tmp dir",
			slog.String("session_id", sessionID.String()),
			slog.Any("error", err),
		)
	}
}

func (s *service) checkExpiry(sess *models.UploadSession) error {
	if time.Now().After(sess.ExpiresAt) {
		return apperror.BadRequest("upload session has expired")
	}
	return nil
}

func (s *service) toResponse(sess *models.UploadSession, chunks []models.UploadChunk) *models.SessionResponse {
	uploaded := make([]int, 0, len(chunks))
	uploadedSet := make(map[int]bool, len(chunks))
	for _, c := range chunks {
		uploaded = append(uploaded, c.ChunkIndex)
		uploadedSet[c.ChunkIndex] = true
	}

	missing := s.missingIndicesFromSet(uploadedSet, sess.TotalChunks)

	return &models.SessionResponse{
		SessionID:      sess.ID.String(),
		ItemID:         sess.ItemID,
		AssetType:      sess.AssetType,
		FileName:       sess.FileName,
		FileSize:       sess.FileSize,
		ChunkSize:      sess.ChunkSize,
		TotalChunks:    sess.TotalChunks,
		UploadedChunks: uploaded,
		MissingChunks:  missing,
		Status:         sess.Status,
		FinalURL:       sess.FinalURL,
		ExpiresAt:      sess.ExpiresAt.Format(time.RFC3339),
	}
}

func (s *service) missingIndices(chunks []models.UploadChunk, total int) []int {
	have := make(map[int]bool, len(chunks))
	for _, c := range chunks {
		have[c.ChunkIndex] = true
	}
	return s.missingIndicesFromSet(have, total)
}

func (s *service) missingIndicesFromSet(have map[int]bool, total int) []int {
	missing := make([]int, 0)
	for i := 0; i < total; i++ {
		if !have[i] {
			missing = append(missing, i)
		}
	}
	return missing
}
