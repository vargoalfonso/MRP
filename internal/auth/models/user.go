package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// User is the GORM model for the public.users table.
//
// id   — BIGINT auto-increment, internal primary key (never exposed to clients).
// uuid — UUID v4, the public-facing identifier returned in API responses.
//
// OAuth columns (provider, provider_id, avatar_url) are nullable so local
// accounts work without OAuth. For OAuth-only accounts, leave password empty.
type User struct {
	ID         int64          `gorm:"primaryKey;autoIncrement"                    json:"-"`
	UUID       string         `gorm:"uniqueIndex;not null;default:uuid_generate_v4()" json:"id"`
	Username   string         `gorm:"uniqueIndex;not null"                        json:"username"`
	Email      string         `gorm:"uniqueIndex;not null"                        json:"email"`
	Password   string         `gorm:"default:''"                                  json:"-"`
	Roles      string         `gorm:"default:'user'"                              json:"roles"`
	EmployeeID *string        `gorm:"index;default:null"                          json:"employee_id,omitempty"`

	// OAuth / SSO support
	Provider   string  `gorm:"not null;default:'local'"    json:"provider"`
	ProviderID *string `gorm:"index;default:null"          json:"provider_id,omitempty"`
	AvatarURL  *string `gorm:"default:null"                json:"avatar_url,omitempty"`

	IsVerified bool `gorm:"default:false" json:"is_verified"`

	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

// HashPassword replaces u.Password with its bcrypt hash.
func (u *User) HashPassword(plain string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	return nil
}

// CheckPassword returns nil when plain matches the stored bcrypt hash.
// Returns an error for OAuth-only accounts (empty password).
func (u *User) CheckPassword(plain string) error {
	if u.Password == "" {
		return bcrypt.ErrHashTooShort
	}
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plain))
}
