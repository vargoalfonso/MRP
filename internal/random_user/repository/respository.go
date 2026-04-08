package repository

import (
	"context"
	"database/sql"
)

type (
	repository struct {
		DB *sql.DB
	}

	IRepository interface {
		funcRepoExample(ctx context.Context) error
	}
)

func NewRepository(db *sql.DB) IRepository {
	return &repository{
		DB: db,
	}
}

func (r *repository) funcRepoExample(ctx context.Context) error {
	//query := sqlc.New(r.DB)

	return nil
}
