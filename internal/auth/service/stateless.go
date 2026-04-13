package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ganasa18/go-template/config"
	"github.com/ganasa18/go-template/internal/auth/models"
	authRepo "github.com/ganasa18/go-template/internal/auth/repository"
	"github.com/ganasa18/go-template/pkg/apperror"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type stateless struct {
	cfg  *config.Config
	repo authRepo.IRepository
}

func newStateless(cfg *config.Config, repo authRepo.IRepository) Authenticator {
	return &stateless{cfg: cfg, repo: repo}
}

func (s *stateless) Mode() string { return "stateless" }

func (s *stateless) Register(ctx context.Context, req models.RegisterRequest) (*models.User, error) {
	return registerUser(ctx, s.repo, req)
}

func (s *stateless) Login(ctx context.Context, req models.LoginRequest) (*models.TokenPair, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		// Return generic message so callers can't enumerate valid emails.
		return nil, apperror.Unauthorized("invalid credentials")
	}
	if err = user.CheckPassword(req.Password); err != nil {
		return nil, apperror.Unauthorized("invalid credentials")
	}

	accessToken, expiresAt, err := s.buildToken(user.UUID, strings.Split(user.Roles, ","))
	if err != nil {
		return nil, err
	}

	return &models.TokenPair{
		AccessToken: accessToken,
		ExpiresAt:   expiresAt,
		TokenType:   "Bearer",
	}, nil
}

func (s *stateless) ValidateAccessToken(_ context.Context, tokenStr string) (*models.Claims, error) {
	return parseToken(tokenStr, s.cfg.JWTAccessSecret)
}

// RefreshTokens is not available in stateless mode.
func (s *stateless) RefreshTokens(_ context.Context, _ string) (*models.TokenPair, error) {
	return nil, apperror.New(405, apperror.CodeForbidden, "refresh tokens are not supported in stateless mode")
}

// RevokeToken is a no-op in stateless mode; tokens expire on their own.
func (s *stateless) RevokeToken(_ context.Context, _ string) error { return nil }

// buildToken signs a new access JWT and returns the token string + expiry.
func (s *stateless) buildToken(userID string, roles []string) (string, time.Time, error) {
	expiresAt := time.Now().Add(s.cfg.JWTAccessTTL)
	claims := models.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			ID:        uuid.New().String(), // jti
		},
		UserID: userID,
		Roles:  roles,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.cfg.JWTAccessSecret))
	if err != nil {
		return "", time.Time{}, apperror.InternalWrap("failed to sign token", err)
	}
	return signed, expiresAt, nil
}

// parseToken is a shared helper used by both stateless and stateful implementations.
func parseToken(tokenStr, secret string) (*models.Claims, error) {
	claims := &models.Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, apperror.TokenInvalid()
		}
		return []byte(secret), nil
	})
	if err != nil {
		if err == jwt.ErrTokenExpired {
			return nil, apperror.TokenExpired()
		}
		return nil, apperror.TokenInvalid()
	}
	if !token.Valid {
		return nil, apperror.TokenInvalid()
	}
	return claims, nil
}

func (s *stateless) SetPassword(ctx context.Context, token string, password string, confirm string) error {

	// ==============================
	// 🔥 VALIDASI PASSWORD MATCH
	// ==============================
	if password != confirm {
		return fmt.Errorf("password dan konfirmasi password tidak sama")
	}

	if len(password) < 6 {
		return fmt.Errorf("password minimal 6 karakter")
	}

	// ==============================
	// 🔥 AMBIL TOKEN
	// ==============================
	act, err := s.repo.FindValidActivation(ctx, token)
	if err != nil {
		return fmt.Errorf("token tidak valid")
	}

	if act.Used {
		return fmt.Errorf("token sudah digunakan")
	}

	if time.Now().After(act.ExpiredAt) {
		return fmt.Errorf("token expired")
	}

	// ==============================
	// 🔥 HASH PASSWORD
	// ==============================
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// ==============================
	// 🔥 UPDATE USER
	// ==============================
	err = s.repo.UpdateUserPassword(ctx, act.UserID, string(hashed))
	if err != nil {
		return err
	}

	// ==============================
	// 🔥 MARK TOKEN USED
	// ==============================
	err = s.repo.MarkTokenUsed(ctx, act.ID)
	if err != nil {
		return err
	}

	return nil
}
