package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"wordsGo/internal/repository/modelsDB"
)

type UserRepo struct {
	db *Postgres
}

var allowedUpdateColumns = map[string]bool{
	"email":         true,
	"password_hash": true,
	"first_name":    true,
	"last_name":     true,
	"role":          true,
}

func NewUserRepo(pg *Postgres) *UserRepo {
	return &UserRepo{db: pg}
}

func (r *UserRepo) Create(ctx context.Context, req *modelsDB.UserDB) (*modelsDB.UserDB, error) {

	query := `
		insert into users 
    	(email, password_hash, first_name, last_name, role, source_lang, target_lang)
		values ($1, $2, $3, $4, $5, $6, $7)
		returning id, email, first_name, last_name, role, source_lang, target_lang, created_at, updated_at`

	var res modelsDB.UserDB
	err := r.db.db.QueryRowxContext(ctx, query,
		req.Email,
		req.PasswordHash,
		req.FirstName,
		req.LastName,
		req.Role,
		req.SourceLang,
		req.TargetLang,
	).StructScan(&res)

	if err != nil {
		return nil, modelsDB.ParseDBError(err)
	}

	return &res, nil
}

func (r *UserRepo) GetPasswordHashByEmail(ctx context.Context, email string) (*modelsDB.UserDB, error) {
	query := `select id, email, password_hash from users where email = $1 and deleted_at is null`

	var userModel modelsDB.UserDB
	err := r.db.db.QueryRowxContext(ctx, query, email).StructScan(&userModel)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, modelsDB.ErrUserNotFound
		}
		return nil, modelsDB.ParseDBError(err)
	}
	return &userModel, nil

}

func (r *UserRepo) GetUserByID(ctx context.Context, id uuid.UUID) (*modelsDB.UserDB, error) {
	query := `select id, email, first_name, last_name, role, source_lang, target_lang, created_at, updated_at from users where id = $1 and deleted_at is null`
	var userModel modelsDB.UserDB
	err := r.db.db.QueryRowxContext(ctx, query, id).StructScan(&userModel)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, modelsDB.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to get user by id: %w", err)
	}
	return &userModel, nil
}

func (r *UserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	query := `update users set deleted_at = now() where id = $1`
	_, err := r.db.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

func (r *UserRepo) Update(ctx context.Context, id uuid.UUID, fields map[string]any) (*modelsDB.UserDB, error) {

	setParts := make([]string, 0, len(fields))
	args := make([]any, 0, len(fields)+1)

	i := 1
	for column, val := range fields {
		if !allowedUpdateColumns[column] {
			return nil, fmt.Errorf("column %s is not allowed", column)
		}
		setParts = append(setParts, fmt.Sprintf("%s = $%d", column, i))
		args = append(args, val)
		i++
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", i))
	args = append(args, time.Now())

	args = append(args, id)

	query := fmt.Sprintf(`
       UPDATE users
       SET %s
       WHERE id = $%d
       RETURNING id, email, password_hash, first_name, last_name, role, source_lang, target_lang, created_at, updated_at
   `, strings.Join(setParts, ", "), i+1)

	var updatedUser modelsDB.UserDB
	err := r.db.db.QueryRowxContext(ctx, query, args...).StructScan(&updatedUser)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, modelsDB.ErrUserNotFound
		}
		//return nil, fmt.Errorf("failed to update user: %w", err)
		return nil, err
	}
	return &updatedUser, nil
}

func (r *UserRepo) GetUsers(ctx context.Context, order string, pagination modelsDB.Pagination) ([]modelsDB.UserDB, uint64, error) {
	sortOrder := "DESC"
	if strings.ToUpper(order) == "ASC" {
		sortOrder = "ASC"
	}
	query := fmt.Sprintf(`select id, email, first_name, last_name, role, source_lang, target_lang, created_at, updated_at, count(id) over() as total from users
                                                               where deleted_at is null
                                                               order by created_at %s
                                                               limit $1 offset $2`, sortOrder)
	var userDBWithTotal []modelsDB.UserDBWithTotal
	err := r.db.db.SelectContext(ctx, &userDBWithTotal, query, pagination.Limit, pagination.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all users: %w", err)
	}

	if len(userDBWithTotal) == 0 {
		return []modelsDB.UserDB{}, 0, nil
	}

	total := userDBWithTotal[0].Total

	usersDB := make([]modelsDB.UserDB, len(userDBWithTotal))
	for i, user := range userDBWithTotal {
		usersDB[i] = user.UserDB
	}

	return usersDB, total, nil
}
