package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"wordsGo/internal/config"
	"wordsGo/internal/models"
	"wordsGo/internal/repository"
	modelsRepo "wordsGo/internal/repository/modelsDB"
	"wordsGo/internal/utils"
	"wordsGo/slogger"
)

type userService struct {
	repo      repository.UserRepository
	jwtSecret string
	jwtTTL    time.Duration
}

func NewUserService(repo repository.UserRepository, jwtSecret string, jwtTTL time.Duration) UserService {
	return &userService{
		repo:      repo,
		jwtSecret: jwtSecret,
		jwtTTL:    jwtTTL,
	}
}

type UserService interface {
	Create(ctx context.Context, req models.CreateUserRequest) (*models.UserResponse, error)
	Authenticate(ctx context.Context, email, password string) (*models.User, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*models.UserResponse, error)
	Delete(ctx context.Context, requester *models.User, targetID uuid.UUID) error
	Update(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) (*models.UserResponse, error)
	GetUsers(ctx context.Context, limit, page uint64, order string) (*models.ListOfUsersResponse, error)
	Login(ctx context.Context, req models.LoginRequest) (string, error)
	SyncAdmin(ctx context.Context, adminCfg config.Admin) error
}

func (s *userService) Create(ctx context.Context, req models.CreateUserRequest) (*models.UserResponse, error) {
	userRequest, err := models.NewUser(req, models.RoleUser)
	if err != nil {
		return nil, fmt.Errorf("invalid user data")
	}

	userDB, err := s.repo.Create(ctx, models.ToUserDB(userRequest))

	if err != nil {
		if errors.Is(err, modelsRepo.ErrDuplicateEmail) {
			return nil, models.ErrUserAlreadyExists
		}
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return models.FromDBToUserResponse(userDB), nil
}

func (s *userService) Login(ctx context.Context, req models.LoginRequest) (string, error) {
	userDB, err := s.repo.GetPasswordHashByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, modelsRepo.ErrUserNotFound) {
			return "", models.ErrInvalidCredentials
		}
		return "", fmt.Errorf("failed to get user by email: %w", err)
	}
	if !utils.ComparePasswords(userDB.PasswordHash, req.Password) {
		return "", models.ErrInvalidCredentials
	}
	token, err := utils.GenerateToken(userDB.ID, userDB.Role, s.jwtSecret, s.jwtTTL)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	return token, nil
}

func (s *userService) SyncAdmin(ctx context.Context, adminCfg config.Admin) error {
	slogger.Log.InfoContext(ctx, "Syncing admin user...", "email", adminCfg.Email)

	userDB, err := s.repo.GetPasswordHashByEmail(ctx, adminCfg.Email)
	if err != nil {
		if errors.Is(err, modelsRepo.ErrUserNotFound) {
			slogger.Log.InfoContext(ctx, "Admin not found, creating new one")
			req := models.CreateUserRequest{
				Email:      adminCfg.Email,
				Password:   adminCfg.Password,
				FirstName:  "Super",
				LastName:   "Admin",
				SourceLang: "ru",
				TargetLang: "en",
			}
			newUser, _ := models.NewUser(req, models.RoleAdmin)
			_, err = s.repo.Create(ctx, models.ToUserDB(newUser))
			return err
		}
		return err
	}

	hash, err := utils.HashPassword(adminCfg.Password)
	if err != nil {
		return err
	}

	fields := map[string]any{
		"password_hash": hash,
		"role":          string(models.RoleAdmin),
	}

	_, err = s.repo.Update(ctx, userDB.ID, fields)
	if err != nil {
		return fmt.Errorf("failed to update admin: %w", err)
	}
	slogger.Log.InfoContext(ctx, "Admin user synced successfully")

	return nil
}

func (s *userService) Authenticate(ctx context.Context, email, password string) (*models.User, error) {
	user, err := s.repo.GetPasswordHashByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	if !utils.ComparePasswords(user.PasswordHash, password) {
		return nil, models.ErrInvalidCredentials
	}
	return models.FromUserDB(user), nil
}

func (s *userService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.UserResponse, error) {
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, modelsRepo.ErrUserNotFound) {
			return nil, models.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to GetUserByID: %w", err)
	}
	return models.FromDBToUserResponse(user), nil
}

func (s *userService) Delete(ctx context.Context, requester *models.User, targetID uuid.UUID) error {
	target, err := s.repo.GetUserByID(ctx, targetID)
	if err != nil {
		if errors.Is(err, modelsRepo.ErrUserNotFound) {
			return models.ErrUserNotFound
		}
		return fmt.Errorf("failed to get user by id: %w", err)
	}

	if !PermissionCheck(requester, target) {
		return models.ErrPermissionDenied
	}
	err = s.repo.Delete(ctx, targetID)
	if err != nil {
		if errors.Is(err, modelsRepo.ErrUserNotFound) {
			return models.ErrUserNotFound
		}
		return fmt.Errorf("failed to delete user: %w", err)
	}
	return nil
}

func (s *userService) Update(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) (*models.UserResponse, error) {

	fields := map[string]any{}
	var err error

	if req.Email != nil {
		fields["email"] = *req.Email
	}
	if req.Password != nil {
		fields["password_hash"], err = utils.HashPassword(*req.Password)
		if err != nil {
			return nil, err
		}
	}
	if req.FirstName != nil {
		fields["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		fields["last_name"] = *req.LastName
	}

	if len(fields) == 0 {
		currentUser, err := s.repo.GetUserByID(ctx, id)
		if err != nil {
			if errors.Is(err, modelsRepo.ErrUserNotFound) {
				return nil, models.ErrUserNotFound
			}
			return nil, fmt.Errorf("failed to get user by id: %w", err)
		}
		return models.FromDBToUserResponse(currentUser), nil
	}
	updatedUser, err := s.repo.Update(ctx, id, fields)
	slogger.Log.DebugContext(ctx, "UpdateUser from repo update", "updatedUser", updatedUser)
	if err != nil {
		if errors.Is(err, modelsRepo.ErrUserNotFound) {
			return nil, models.ErrUserNotFound
		}
		return nil, fmt.Errorf("failed to update user s.repo.UpdateUser: %w", err)
	}
	return models.FromDBToUserResponse(updatedUser), nil
}

func (s *userService) GetUsers(ctx context.Context, limit, page uint64, order string) (*models.ListOfUsersResponse, error) {

	if limit == 0 {
		limit = 10
	}
	if page == 0 {
		page = 1
	}

	offset := (page - 1) * limit
	pagination := &modelsRepo.Pagination{
		Limit:  limit,
		Offset: offset,
	}
	usersDB, total, err := s.repo.GetUsers(ctx, order, *pagination)
	if err != nil {
		return nil, err
	}

	usersResponse := make([]*models.UserResponse, len(usersDB))
	for i, userModel := range usersDB {
		usersResponse[i] = models.FromDBToUserResponse(&userModel)
	}

	pages := (total + limit - 1) / limit

	resp := &models.ListOfUsersResponse{
		Page:  page,
		Limit: limit,
		Total: total,
		Pages: pages,
		Data:  usersResponse,
	}

	return resp, nil
}

func PermissionCheck(requester *models.User, target *modelsRepo.UserDB) bool {
	if requester.Role == models.RoleAdmin {
		return true
	}
	if requester.ID == target.ID {
		return true
	}
	if requester.Role == models.RoleModerator {
		return target.Role == string(models.RoleUser)
	}
	return false
}
