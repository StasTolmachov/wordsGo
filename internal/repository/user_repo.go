package repository

import (
	"context"

	"github.com/google/uuid"

	"wordsGo/internal/repository/modelsDB"
)

type UserRepository interface {
	Create(ctx context.Context, req *modelsDB.UserDB) (*modelsDB.UserDB, error)
	GetPasswordHashByEmail(ctx context.Context, email string) (*modelsDB.UserDB, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*modelsDB.UserDB, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, fields map[string]any) (*modelsDB.UserDB, error)
	GetUsers(ctx context.Context, order string, pagination modelsDB.Pagination) ([]modelsDB.UserDB, uint64, error)
}
