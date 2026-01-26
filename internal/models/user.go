package models

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"wordsGo/internal/repository/modelsDB"
	"wordsGo/internal/utils"
)

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
	Role         UserRole
	SourceLang   string
	TargetLang   string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

// NewUser creates a new User instance from the provided CreateUserRequest, validating required fields and hashing the password.
func NewUser(req CreateUserRequest, role UserRole) (*User, error) {
	if req.Email == "" || req.Password == "" || req.SourceLang == "" || req.TargetLang == "" {
		return nil, fmt.Errorf("cannot create user with empty fields")
	}
	err := utils.ValidatePassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	hash, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("cannot hash password: %w", err)
	}

	id := uuid.New()

	timeNow := time.Now()

	return &User{
		ID:           id,
		Email:        req.Email,
		PasswordHash: hash,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Role:         role,
		SourceLang:   req.SourceLang,
		TargetLang:   req.TargetLang,
		CreatedAt:    timeNow,
		UpdatedAt:    timeNow,
		DeletedAt:    nil,
	}, nil
}

// ToUserResponse converts a User domain object into a UserResponse DTO for external use.
func ToUserResponse(user *User) *UserResponse {
	return &UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
		UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
	}
}

// FromUserDB converts a UserDB data model to a User domain object.
func FromUserDB(user *modelsDB.UserDB) *User {
	return &User{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Role:         UserRole(user.Role),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}
func ToUserDB(user *User) *modelsDB.UserDB {
	return &modelsDB.UserDB{
		ID:           user.ID,
		Email:        user.Email,
		PasswordHash: user.PasswordHash,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Role:         string(user.Role),
		SourceLang:   user.SourceLang,
		TargetLang:   user.TargetLang,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		DeletedAt:    user.DeletedAt,
	}
}

// CreateUserRequest represents a request payload for creating a new user with required fields and validation rules.
type CreateUserRequest struct {
	Email      string `json:"email" validate:"required,email"`
	Password   string `json:"password" validate:"required,min=8"`
	FirstName  string `json:"first_name" validate:"required"`
	LastName   string `json:"last_name" validate:"required"`
	SourceLang string `json:"source_lang" validate:"required"`
	TargetLang string `json:"target_lang" validate:"required"`
}

// UpdateUserRequest represents a request to update a user's information, with optional fields for partial updates.
type UpdateUserRequest struct {
	Email     *string `json:"email,omitempty"`
	Password  *string `json:"password,omitempty"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
}

type UserResponse struct {
	ID         string `json:"id"`
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	Role       string `json:"role"`
	SourceLang string `json:"source_lang"`
	TargetLang string `json:"target_lang"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

func FromDBToUserResponse(user *modelsDB.UserDB) *UserResponse {
	return &UserResponse{
		ID:         user.ID.String(),
		Email:      user.Email,
		FirstName:  user.FirstName,
		LastName:   user.LastName,
		Role:       user.Role,
		SourceLang: user.SourceLang,
		TargetLang: user.TargetLang,
		CreatedAt:  user.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  user.UpdatedAt.Format(time.RFC3339),
	}
}

type ListOfUsersResponse struct {
	Page  uint64          `json:"page"`
	Limit uint64          `json:"limit"`
	Total uint64          `json:"total"`
	Pages uint64          `json:"pages"`
	Data  []*UserResponse `json:"data"`
}

var (
	ErrUserAlreadyExists  = errors.New("user already exists")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrPermissionDenied   = errors.New("permission denied")
)

type UserRole string

const (
	RoleUser      UserRole = "user"
	RoleModerator UserRole = "moderator"
	RoleAdmin     UserRole = "admin"
)

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
}
