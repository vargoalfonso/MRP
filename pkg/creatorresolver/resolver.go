// Package creatorresolver resolves a user UUID string into a (*uuid.UUID, *string)
// pair suitable for CreatedBy / Validator columns across all service modules.
package creatorresolver

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Resolve looks up the username for userUUID in the users table.
// Returns (parsedUUID, name, nil) on success.
// Falls back gracefully without returning an error when the user is not found.
func Resolve(ctx context.Context, db *gorm.DB, userUUID string) (*uuid.UUID, *string, error) {
	userUUID = strings.TrimSpace(userUUID)
	if userUUID == "" || userUUID == "system" {
		n := "system"
		return nil, &n, nil
	}

	parsed, err := uuid.Parse(userUUID)
	if err != nil {
		// Not a valid UUID – store raw string as name; created_by stays NULL.
		n := userUUID
		return nil, &n, nil
	}

	var username string
	q := db.WithContext(ctx).Table("users").Select("username").Where("uuid = ?", userUUID).Limit(1)
	if err := q.Take(&username).Error; err != nil {
		n := userUUID
		return &parsed, &n, nil
	}
	if strings.TrimSpace(username) == "" {
		n := userUUID
		return &parsed, &n, nil
	}
	n := username
	return &parsed, &n, nil
}
