package service

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"wordsGo/internal/config"
	"wordsGo/internal/models"
	mocks "wordsGo/internal/repository/mocks"
	modelsRepo "wordsGo/internal/repository/modelsDB"
	"wordsGo/internal/utils"
	"wordsGo/slogger"
)

func TestMain(m *testing.M) {
	slogger.MakeLogger(true)
	code := m.Run()
	os.Exit(code)
}

var jwtSecret string = "secret"
var jwtTTL time.Duration = 10 * time.Minute

func TestUserService_Create(t *testing.T) {
	ctx := t.Context()
	req := models.CreateUserRequest{
		Email:     "test@example.com",
		Password:  "StrongPass1!",
		FirstName: "John",
		LastName:  "Doe",
	}

	tests := []struct {
		name          string
		mockBehavior  func(r *mocks.MockUserRepository)
		expectedError error
	}{
		{
			name: "success",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("CreateUser", ctx, mock.AnythingOfType("*modelsDB.UserDB")).Return(&modelsRepo.UserDB{
					ID:        uuid.New(),
					Email:     req.Email,
					FirstName: req.FirstName,
					LastName:  req.LastName,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				}, nil)
			},
			expectedError: nil,
		},
		{
			name: "Duplicate Email",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("CreateUser", ctx, mock.AnythingOfType("*modelsDB.UserDB")).Return(nil, modelsRepo.ErrDuplicateEmail)
			},
			expectedError: models.ErrUserAlreadyExists,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository(t)
			tt.mockBehavior(mockRepo)
			service := NewUserService(mockRepo, jwtSecret, jwtTTL)
			resp, err := service.Create(ctx, req)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, req.Email, resp.Email)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_Authenticate(t *testing.T) {
	ctx := t.Context()
	email := "user25@com"
	password := "Password1!"
	passwordHash := "$2a$10$3oKmVShrERUMr2pFumIYuOaCJj3iEMFvDLf1//OwuvBEuGlv0y.QO"

	tests := []struct {
		name          string
		mockBehavior  func(r *mocks.MockUserRepository)
		expectedError error
	}{
		{
			name: "success",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetPasswordHashByEmail", ctx, email).Return(&modelsRepo.UserDB{
					ID:           uuid.New(),
					Email:        email,
					PasswordHash: "$2a$10$3oKmVShrERUMr2pFumIYuOaCJj3iEMFvDLf1//OwuvBEuGlv0y.QO",
					FirstName:    "John",
					LastName:     "Doe",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}, nil)
			},
			expectedError: nil,
		},
		{
			name: "ErrUserNotFound",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetPasswordHashByEmail", ctx, email).Return(nil, modelsRepo.ErrUserNotFound)
			},
			expectedError: modelsRepo.ErrUserNotFound,
		},
		{
			name: "invalid credentials",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetPasswordHashByEmail", ctx, email).Return(&modelsRepo.UserDB{
					ID:           uuid.New(),
					Email:        email,
					PasswordHash: "StrongPass1!",
					FirstName:    "John",
					LastName:     "Doe",
					CreatedAt:    time.Now(),
					UpdatedAt:    time.Now(),
				}, nil)
			},
			expectedError: models.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository(t)
			tt.mockBehavior(mockRepo)
			service := NewUserService(mockRepo, jwtSecret, jwtTTL)

			resp, err := service.Authenticate(ctx, email, password)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, resp)

			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.Equal(t, passwordHash, resp.PasswordHash)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetUserByID(t *testing.T) {
	ctx := t.Context()
	userID := uuid.New()
	unexpectedErr := errors.New("unexpected error")
	repoResult := &modelsRepo.UserDB{
		ID:           userID,
		Email:        "user25@com",
		PasswordHash: "StrongPass1!",
		FirstName:    "John",
		LastName:     "Doe",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	expectedResponse := models.FromDBToUserResponse(repoResult)

	tests := []struct {
		name          string
		mockBehavior  func(r *mocks.MockUserRepository)
		expectedError error
		expectedBody  *models.UserResponse
	}{
		{
			name: "success",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetUserByID", ctx, userID).Return(repoResult, nil)
			},
			expectedError: nil,
			expectedBody:  expectedResponse,
		},
		{
			name: "ErrUserNotFound",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetUserByID", ctx, userID).Return(nil, modelsRepo.ErrUserNotFound)
			},
			expectedError: models.ErrUserNotFound,
			expectedBody:  nil,
		},
		{
			name: "Failure: Unexpected Repo Error",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetUserByID", ctx, userID).Return(nil, unexpectedErr)
			},
			expectedError: unexpectedErr,
			expectedBody:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository(t)
			tt.mockBehavior(mockRepo)
			service := NewUserService(mockRepo, jwtSecret, jwtTTL)
			resp, err := service.GetUserByID(ctx, userID)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedBody, resp)
			}
			mockRepo.AssertExpectations(t)
		})
	}

}

func TestUserService_Delete(t *testing.T) {
	ctx := t.Context()
	userID := uuid.New()
	unexpectedErr := errors.New("unexpected error")
	requester := &models.User{ID: userID}

	tests := []struct {
		name          string
		mockBehavior  func(r *mocks.MockUserRepository)
		expectedError error
		requester     *models.User
	}{
		{
			name: "success",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetUserByID", ctx, userID).Return(&modelsRepo.UserDB{ID: userID}, nil)
				r.On("DeleteUser", ctx, userID).Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "ErrUserNotFound",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetUserByID", ctx, userID).Return(&modelsRepo.UserDB{ID: userID}, nil)
				r.On("DeleteUser", ctx, userID).Return(modelsRepo.ErrUserNotFound)
			},
			expectedError: models.ErrUserNotFound,
		},
		{
			name: "Failure: Unexpected Repo Error",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetUserByID", ctx, userID).Return(&modelsRepo.UserDB{ID: userID}, nil)
				r.On("DeleteUser", ctx, userID).Return(unexpectedErr)
			},
			expectedError: unexpectedErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository(t)
			tt.mockBehavior(mockRepo)
			service := NewUserService(mockRepo, jwtSecret, jwtTTL)
			err := service.Delete(ctx, requester, userID)
			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_Update(t *testing.T) {
	ctx := t.Context()
	userID := uuid.New()
	unexpectedErr := errors.New("unexpected repo error")

	initialUserDB := &modelsRepo.UserDB{
		ID:           userID,
		Email:        "old@example.com",
		PasswordHash: "old_hash",
		FirstName:    "Old",
		LastName:     "User",
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	tests := []struct {
		name          string
		req           models.UpdateUserRequest
		mockBehavior  func(r *mocks.MockUserRepository)
		expectedError error
		assertFunc    func(t *testing.T, resp *models.UserResponse)
	}{
		{
			name: "Success: UpdateUser Email Only",
			req: models.UpdateUserRequest{
				Email: ptr("new@example.com"),
			},
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("UpdateUser", mock.Anything, userID, mock.IsType(map[string]any{})).
					Run(func(args mock.Arguments) {
						fields := args.Get(2).(map[string]any)
						assert.Contains(t, fields, "email")
						assert.NotContains(t, fields, "password_hash")
					}).
					Return(&modelsRepo.UserDB{ID: userID, Email: "new@example.com"}, nil)
			},
			expectedError: nil,
			assertFunc: func(t *testing.T, resp *models.UserResponse) {
				assert.Equal(t, "new@example.com", resp.Email)
			},
		},
		{
			name: "Success: UpdateUser Password Only (check hash in fields)",
			req: models.UpdateUserRequest{
				Password: ptr("NewPass123!"),
			},
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("UpdateUser", mock.Anything, userID, mock.IsType(map[string]any{})).
					Run(func(args mock.Arguments) {
						fields := args.Get(2).(map[string]any)
						hash, ok := fields["password_hash"].(string)
						assert.True(t, ok)
						assert.Greater(t, len(hash), 10)
					}).
					Return(&modelsRepo.UserDB{ID: userID, PasswordHash: "fake_hash_ok"}, nil)
			},
			expectedError: nil,
		},
		{
			name: "Success: No Fields To UpdateUser (No-op)",
			req:  models.UpdateUserRequest{},
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetUserByID", mock.Anything, userID).Return(initialUserDB, nil)
				r.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything, mock.Anything)
			},
			expectedError: nil,
			assertFunc: func(t *testing.T, resp *models.UserResponse) {
				assert.Equal(t, initialUserDB.Email, resp.Email)
			},
		},
		{
			name: "Failure: User Not Found",
			req: models.UpdateUserRequest{
				FirstName: ptr("Test"),
			},
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("UpdateUser", mock.Anything, userID, mock.Anything).Return(nil, modelsRepo.ErrUserNotFound)
			},
			expectedError: models.ErrUserNotFound,
		},
		{
			name: "Failure: Unexpected Repository Error",
			req: models.UpdateUserRequest{
				LastName: ptr("Test"),
			},
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("UpdateUser", mock.Anything, userID, mock.Anything).Return(nil, unexpectedErr)
			},
			expectedError: unexpectedErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository(t)
			tt.mockBehavior(mockRepo)
			service := NewUserService(mockRepo, jwtSecret, jwtTTL)

			resp, err := service.Update(ctx, userID, tt.req)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				if tt.assertFunc != nil {
					tt.assertFunc(t, resp)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_GetUsers(t *testing.T) {
	ctx := t.Context()
	unexpectedErr := errors.New("database connection failed")

	user1 := modelsRepo.UserDB{ID: uuid.New(), Email: "u1@e.com", FirstName: "A", LastName: "A", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	user2 := modelsRepo.UserDB{ID: uuid.New(), Email: "u2@e.com", FirstName: "B", LastName: "B", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	user3 := modelsRepo.UserDB{ID: uuid.New(), Email: "u3@e.com", FirstName: "C", LastName: "C", CreatedAt: time.Now(), UpdatedAt: time.Now()}

	usersDB := []modelsRepo.UserDB{user1, user2, user3}
	usersResp := []*models.UserResponse{
		models.FromDBToUserResponse(&user1),
		models.FromDBToUserResponse(&user2),
		models.FromDBToUserResponse(&user3),
	}

	tests := []struct {
		name          string
		limit         uint64
		page          uint64
		order         string
		mockBehavior  func(r *mocks.MockUserRepository)
		expectedError error
		expectedBody  *models.ListOfUsersResponse
	}{
		{
			name:  "Success: Default Pagination (limit=10, page=1)",
			limit: 0,
			page:  0,
			order: "desc",
			mockBehavior: func(r *mocks.MockUserRepository) {
				expectedPagination := modelsRepo.Pagination{Limit: 10, Offset: 0}
				r.On("GetUsers", mock.Anything, "desc", expectedPagination).
					Return(usersDB, uint64(25), nil)
			},
			expectedError: nil,
			expectedBody: &models.ListOfUsersResponse{
				Page:  1,
				Limit: 10,
				Total: 25,
				Pages: 3,
				Data:  usersResp,
			},
		},
		{
			name:  "Success: Custom Pagination (limit=5, page=2)",
			limit: 5,
			page:  2,
			order: "asc",
			mockBehavior: func(r *mocks.MockUserRepository) {
				expectedPagination := modelsRepo.Pagination{Limit: 5, Offset: 5}
				r.On("GetUsers", mock.Anything, "asc", expectedPagination).
					Return(usersDB, uint64(12), nil)
			},
			expectedError: nil,
			expectedBody: &models.ListOfUsersResponse{
				Page:  2,
				Limit: 5,
				Total: 12,
				Pages: 3,
				Data:  usersResp,
			},
		},
		{
			name:  "Success: No Users Found",
			limit: 10,
			page:  1,
			order: "desc",
			mockBehavior: func(r *mocks.MockUserRepository) {
				expectedPagination := modelsRepo.Pagination{Limit: 10, Offset: 0}
				r.On("GetUsers", mock.Anything, "desc", expectedPagination).
					Return([]modelsRepo.UserDB{}, uint64(0), nil)
			},
			expectedError: nil,
			expectedBody: &models.ListOfUsersResponse{
				Page:  1,
				Limit: 10,
				Total: 0,
				Pages: 0,
				Data:  []*models.UserResponse{},
			},
		},
		{
			name:  "Failure: Unexpected Repo Error",
			limit: 10,
			page:  1,
			order: "desc",
			mockBehavior: func(r *mocks.MockUserRepository) {
				expectedPagination := modelsRepo.Pagination{Limit: 10, Offset: 0}
				r.On("GetUsers", mock.Anything, "desc", expectedPagination).
					Return(nil, uint64(0), unexpectedErr)
			},
			expectedError: unexpectedErr,
			expectedBody:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository(t)
			tt.mockBehavior(mockRepo)
			service := NewUserService(mockRepo, jwtSecret, jwtTTL)

			resp, err := service.GetUsers(ctx, tt.limit, tt.page, tt.order)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, resp)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, resp)

				assert.Equal(t, tt.expectedBody.Page, resp.Page, "Page mismatch")
				assert.Equal(t, tt.expectedBody.Limit, resp.Limit, "Limit mismatch")
				assert.Equal(t, tt.expectedBody.Total, resp.Total, "Total mismatch")
				assert.Equal(t, tt.expectedBody.Pages, resp.Pages, "Pages mismatch")

				assert.Equal(t, len(tt.expectedBody.Data), len(resp.Data), "Data length mismatch")
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func ptr[T any](v T) *T {
	return &v
}

func TestPermissionCheck(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()

	tests := []struct {
		name      string
		requester *models.User
		target    *modelsRepo.UserDB
		want      bool
	}{
		{"Admin deletes anyone", &models.User{Role: models.RoleAdmin}, &modelsRepo.UserDB{Role: "user"}, true},
		{"User deletes self", &models.User{ID: id1, Role: models.RoleUser}, &modelsRepo.UserDB{ID: id1, Role: "user"}, true},
		{"User deletes other", &models.User{ID: id1, Role: models.RoleUser}, &modelsRepo.UserDB{ID: id2, Role: "user"}, false},
		{"Moderator deletes User", &models.User{Role: models.RoleModerator}, &modelsRepo.UserDB{Role: "user"}, true},
		{"Moderator deletes Admin", &models.User{ID: id1, Role: models.RoleModerator}, &modelsRepo.UserDB{ID: id2, Role: "admin"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, PermissionCheck(tt.requester, tt.target))
		})
	}
}

func TestUserService_Login(t *testing.T) {
	ctx := t.Context()
	email := "test@example.com"
	correctPassword := "StrongPass1!"

	validHash, err := utils.HashPassword(correctPassword)
	assert.NoError(t, err)

	userID := uuid.New()

	tests := []struct {
		name          string
		inputEmail    string
		inputPassword string
		mockBehavior  func(r *mocks.MockUserRepository)
		expectedToken bool
		expectedError error
	}{
		{
			name:          "Success",
			inputEmail:    email,
			inputPassword: correctPassword,
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetPasswordHashByEmail", ctx, email).Return(&modelsRepo.UserDB{
					ID:           userID,
					Email:        email,
					PasswordHash: validHash,
					Role:         "user",
				}, nil)
			},
			expectedToken: true,
			expectedError: nil,
		},
		{
			name:          "Failure: User Not Found",
			inputEmail:    email,
			inputPassword: correctPassword,
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetPasswordHashByEmail", ctx, email).Return(nil, modelsRepo.ErrUserNotFound)
			},
			expectedToken: false,
			expectedError: models.ErrInvalidCredentials,
		},
		{
			name:          "Failure: Wrong Password",
			inputEmail:    email,
			inputPassword: "wrong_password",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetPasswordHashByEmail", ctx, email).Return(&modelsRepo.UserDB{
					ID:           userID,
					Email:        email,
					PasswordHash: validHash,
					Role:         "user",
				}, nil)
			},
			expectedToken: false,
			expectedError: models.ErrInvalidCredentials,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository(t)
			tt.mockBehavior(mockRepo)

			service := NewUserService(mockRepo, jwtSecret, jwtTTL)

			req := models.LoginRequest{
				Email:    tt.inputEmail,
				Password: tt.inputPassword,
			}
			token, err := service.Login(ctx, req)

			if tt.expectedError != nil {
				assert.ErrorIs(t, err, tt.expectedError)
				assert.Empty(t, token)
			} else {
				assert.NoError(t, err)
				if tt.expectedToken {
					assert.NotEmpty(t, token)
				}
			}

			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUserService_SyncAdmin(t *testing.T) {
	ctx := t.Context()
	adminCfg := config.Admin{
		Email:    "admin@example.com",
		Password: "SecureAdminPassword1!",
	}
	existingAdminID := uuid.New()
	unexpectedErr := errors.New("db error")

	tests := []struct {
		name          string
		mockBehavior  func(r *mocks.MockUserRepository)
		expectedError error
	}{
		{
			name: "Success: Admin not found, create new",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetPasswordHashByEmail", ctx, adminCfg.Email).
					Return(nil, modelsRepo.ErrUserNotFound)

				r.On("CreateUser", ctx, mock.MatchedBy(func(u *modelsRepo.UserDB) bool {
					return u.Email == adminCfg.Email &&
						u.Role == string(models.RoleAdmin) &&
						u.FirstName == "Super" &&
						u.LastName == "Admin" &&
						u.PasswordHash != ""
				})).Return(&modelsRepo.UserDB{
					ID:    uuid.New(),
					Email: adminCfg.Email,
					Role:  string(models.RoleAdmin),
				}, nil)
			},
			expectedError: nil,
		},
		{
			name: "Success: Admin exists, update password and role",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetPasswordHashByEmail", ctx, adminCfg.Email).
					Return(&modelsRepo.UserDB{
						ID:    existingAdminID,
						Email: adminCfg.Email,
						Role:  "user",
					}, nil)

				r.On("UpdateUser", ctx, existingAdminID, mock.MatchedBy(func(fields map[string]any) bool {
					role, roleOk := fields["role"]
					hash, hashOk := fields["password_hash"]

					return roleOk && role == string(models.RoleAdmin) &&
						hashOk && len(hash.(string)) > 0
				})).Return(&modelsRepo.UserDB{
					ID: existingAdminID,
				}, nil)
			},
			expectedError: nil,
		},
		{
			name: "Failure: GetPasswordHashByEmail Error",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetPasswordHashByEmail", ctx, adminCfg.Email).
					Return(nil, unexpectedErr)
			},
			expectedError: unexpectedErr,
		},
		{
			name: "Failure: CreateUser Error",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetPasswordHashByEmail", ctx, adminCfg.Email).
					Return(nil, modelsRepo.ErrUserNotFound)

				r.On("CreateUser", ctx, mock.Anything).
					Return(nil, unexpectedErr)
			},
			expectedError: unexpectedErr,
		},
		{
			name: "Failure: UpdateUser Error",
			mockBehavior: func(r *mocks.MockUserRepository) {
				r.On("GetPasswordHashByEmail", ctx, adminCfg.Email).
					Return(&modelsRepo.UserDB{ID: existingAdminID}, nil)

				r.On("UpdateUser", ctx, existingAdminID, mock.Anything).
					Return(nil, unexpectedErr)
			},
			expectedError: unexpectedErr,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockRepo := mocks.NewMockUserRepository(t)
			tt.mockBehavior(mockRepo)
			service := NewUserService(mockRepo, jwtSecret, jwtTTL)

			err := service.SyncAdmin(ctx, adminCfg)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.ErrorIs(t, err, tt.expectedError)
			} else {
				assert.NoError(t, err)
			}
			mockRepo.AssertExpectations(t)
		})
	}
}
