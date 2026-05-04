package bulkimport

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

const defaultTTL = 30 * time.Minute

// ErrorStore saves error Excel files keyed by a short-lived UUID token and
// serves them for download. Files are deleted after first read or after TTL.
//
// The default implementation uses the OS temp directory; it is safe for
// single-process deployments. For multi-instance deployments replace with a
// Redis-backed store.
type ErrorStore interface {
	// Save stores the Excel bytes and returns a unique download token.
	Save(data []byte) (token string, err error)
	// Get retrieves and deletes the stored file. Returns nil, nil if not found
	// or expired.
	Get(token string) ([]byte, error)
}

// NewFileStore creates an ErrorStore that persists files under dir.
// Pass an empty string to use os.TempDir()/bom_import_errors.
func NewFileStore(dir string) (ErrorStore, error) {
	if dir == "" {
		dir = filepath.Join(os.TempDir(), "bom_import_errors")
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("bulkimport: create store dir: %w", err)
	}
	return &fileStore{dir: dir}, nil
}

type fileStore struct {
	dir string
	mu  sync.Mutex
}

func (s *fileStore) Save(data []byte) (string, error) {
	token := uuid.New().String()
	path := filepath.Join(s.dir, token+".xlsx")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", fmt.Errorf("bulkimport: save error file: %w", err)
	}
	return token, nil
}

func (s *fileStore) Get(token string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Sanitise token — must be a valid UUID (no path traversal)
	if _, err := uuid.Parse(token); err != nil {
		return nil, nil
	}

	path := filepath.Join(s.dir, token+".xlsx")
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("bulkimport: stat error file: %w", err)
	}

	// Enforce TTL
	if time.Since(info.ModTime()) > defaultTTL {
		_ = os.Remove(path)
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("bulkimport: read error file: %w", err)
	}

	// Delete after first successful read
	_ = os.Remove(path)
	return data, nil
}

// ToBytes serialises an excelize.File to []byte via a buffer.
func ToBytes(f interface {
	WriteTo(*bytes.Buffer) (int64, error)
}) ([]byte, error) {
	var buf bytes.Buffer
	if _, err := f.WriteTo(&buf); err != nil {
		return nil, fmt.Errorf("bulkimport: serialise excel: %w", err)
	}
	return buf.Bytes(), nil
}
