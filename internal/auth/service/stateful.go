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
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"
)

// Redis key schema:
//   refresh:{jti}  →  "{userID}"   TTL = JWTRefreshTTL
//   revoked:{jti}  →  "1"          TTL = JWTAccessTTL  (for revoked access tokens)

type stateful struct {
	cfg  *config.Config
	repo authRepo.IRepository
	rdb  *redis.Client
}

func newStateful(cfg *config.Config, repo authRepo.IRepository, rdb *redis.Client) Authenticator {
	return &stateful{cfg: cfg, repo: repo, rdb: rdb}
}

func (s *stateful) Mode() string { return "stateful" }

func (s *stateful) Register(ctx context.Context, req models.RegisterRequest) (*models.User, error) {
	return registerUser(ctx, s.repo, req)
}

func (s *stateful) Login(ctx context.Context, req models.LoginRequest) (*models.TokenPair, error) {
	user, err := s.repo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, apperror.Unauthorized("invalid credentials")
	}
	if err = user.CheckPassword(req.Password); err != nil {
		return nil, apperror.Unauthorized("invalid credentials")
	}

	return s.issueTokenPair(ctx, user.UUID, strings.Split(user.Roles, ","))
}

func (s *stateful) ValidateAccessToken(ctx context.Context, tokenStr string) (*models.Claims, error) {
	claims, err := parseToken(tokenStr, s.cfg.JWTAccessSecret)
	if err != nil {
		return nil, err
	}

	// Check whether the token has been explicitly revoked.
	revoked, err := s.rdb.Exists(ctx, revokedKey(claims.ID)).Result()
	if err != nil {
		return nil, apperror.InternalWrap("redis check failed", err)
	}
	if revoked > 0 {
		return nil, apperror.TokenInvalid()
	}
	return claims, nil
}

func (s *stateful) RefreshTokens(ctx context.Context, refreshToken string) (*models.TokenPair, error) {
	claims, err := parseToken(refreshToken, s.cfg.JWTRefreshSecret)
	if err != nil {
		return nil, err
	}

	// Verify the refresh token is still registered in Redis.
	key := refreshKey(claims.ID)
	userID, err := s.rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		return nil, apperror.TokenInvalid()
	}
	if err != nil {
		return nil, apperror.InternalWrap("redis get failed", err)
	}
	if userID != claims.UserID {
		return nil, apperror.TokenInvalid()
	}

	// Invalidate the old refresh token (rotation).
	s.rdb.Del(ctx, key)

	// Issue a fresh pair.
	user, err := s.repo.FindByUUID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.issueTokenPair(ctx, user.UUID, strings.Split(user.Roles, ","))
}

func (s *stateful) RevokeToken(ctx context.Context, jti string) error {
	err := s.rdb.Set(ctx, revokedKey(jti), "1", s.cfg.JWTAccessTTL).Err()
	if err != nil {
		return apperror.InternalWrap("failed to revoke token", err)
	}
	return nil
}

// issueTokenPair creates a new access + refresh token pair and registers the
// refresh token in Redis.
func (s *stateful) issueTokenPair(ctx context.Context, userID string, roles []string) (*models.TokenPair, error) {
	now := time.Now()
	accessJTI := uuid.New().String()
	refreshJTI := uuid.New().String()

	// Access token (short-lived).
	accessExp := now.Add(s.cfg.JWTAccessTTL)
	accessClaims := models.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(accessExp),
			ID:        accessJTI,
		},
		UserID: userID,
		Roles:  roles,
	}
	accessToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims).
		SignedString([]byte(s.cfg.JWTAccessSecret))
	if err != nil {
		return nil, apperror.InternalWrap("failed to sign access token", err)
	}

	// Refresh token (long-lived, different secret).
	refreshExp := now.Add(s.cfg.JWTRefreshTTL)
	refreshClaims := models.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(refreshExp),
			ID:        refreshJTI,
		},
		UserID: userID,
		Roles:  roles,
	}
	refreshToken, err := jwt.NewWithClaims(jwt.SigningMethodHS256, refreshClaims).
		SignedString([]byte(s.cfg.JWTRefreshSecret))
	if err != nil {
		return nil, apperror.InternalWrap("failed to sign refresh token", err)
	}

	// Register refresh token in Redis with TTL.
	if err = s.rdb.Set(ctx, refreshKey(refreshJTI), userID, s.cfg.JWTRefreshTTL).Err(); err != nil {
		return nil, apperror.InternalWrap("failed to store refresh token", err)
	}

	return &models.TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    accessExp,
		TokenType:    "Bearer",
	}, nil
}

func refreshKey(jti string) string { return fmt.Sprintf("refresh:%s", jti) }
func revokedKey(jti string) string { return fmt.Sprintf("revoked:%s", jti) }

func (s *stateful) SetPassword(ctx context.Context, token string, password string, confirm string) error {

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
