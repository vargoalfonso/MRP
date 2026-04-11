// Package auth provides utilities for extracting user context from JWT tokens.
package auth

import (
	"context"

	authModels "github.com/ganasa18/go-template/internal/auth/models"
	"github.com/ganasa18/go-template/internal/base/app"
	"gorm.io/gorm"
)

// UserContext holds user information extracted from JWT claims.
type UserContext struct {
	UserID   string // UUID from token
	Username string // username (may be empty if not loaded)
	Email    string // email (may be empty if not loaded)
}

// MustExtractUserContext extracts user info from JWT claims.
// Returns a UserContext with at least UserID populated.
// Defaults to "system" user if claims not found.
func MustExtractUserContext(ctx *app.Context) *UserContext {
	raw, exists := ctx.Get("claims")
	if !exists {
		return &UserContext{UserID: "system", Username: "system", Email: "system@erp"}
	}

	claims, ok := raw.(*authModels.Claims)
	if !ok || claims == nil {
		return &UserContext{UserID: "system", Username: "system", Email: "system@erp"}
	}

	userID := claims.UserID
	if userID == "" {
		userID = "system"
	}

	return &UserContext{
		UserID:   userID,
		Username: "", // will be populated if LoadFullUser() is called
		Email:    "",
	}
}

// LoadFullUser queries the database to populate Username and Email.
// Safe to call even if database fails — returns existing UserContext.
func (u *UserContext) LoadFullUser(ctx context.Context, db *gorm.DB) *UserContext {
	if u.UserID == "system" {
		u.Username = "system"
		u.Email = "system@erp"
		return u
	}

	var user authModels.User
	if err := db.WithContext(ctx).
		Select("username, email").
		Where("uuid = ?", u.UserID).
		First(&user).Error; err != nil {
		// If query fails, keep existing values
		if u.Username == "" {
			u.Username = u.UserID // fallback to UUID
		}
		return u
	}

	u.Username = user.Username
	u.Email = user.Email
	return u
}

// GetPerformedBy returns the user identifier for audit fields.
// Prefers Username if loaded, otherwise returns UserID.
func (u *UserContext) GetPerformedBy() string {
	if u.Username != "" {
		return u.Username
	}
	return u.UserID
}
