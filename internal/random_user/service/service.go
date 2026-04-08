package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/ganasa18/go-template/internal/random_user/models"
	randomUserRepository "github.com/ganasa18/go-template/internal/random_user/repository"
	"github.com/ganasa18/go-template/pkg/httpclient"
	"github.com/ganasa18/go-template/pkg/logger"
)

// Service defines the random_user business-logic contract.
type Service interface {
	GetRandomDataUser(ctx context.Context) (models.RandomUser, int, error)
}

type service struct {
	repo       randomUserRepository.IRepository
	httpClient httpclient.Client
}

// NewService constructs a random_user Service.
func NewService(repo randomUserRepository.IRepository, httpClient httpclient.Client) Service {
	return &service{repo: repo, httpClient: httpClient}
}

func (s *service) GetRandomDataUser(ctx context.Context) (models.RandomUser, int, error) {
	var raw interface{}
	url := "https://randomuser.me/api"

	statusCode, err := s.httpClient.GetJSON(ctx, url, nil, &raw)
	if err != nil {
		logger.FromContext(ctx).Error("GetRandomDataUser HTTP call failed",
			slog.String("url", url),
			slog.Any("error", err),
		)
		return models.RandomUser{}, statusCode, fmt.Errorf("GetRandomDataUser: %w", err)
	}

	b, _ := json.Marshal(raw)
	var result models.RandomUser
	_ = json.Unmarshal(b, &result)

	return result, statusCode, nil
}
