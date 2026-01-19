package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"wordsGo/internal/middleware"
	"wordsGo/internal/models"
	mocks "wordsGo/internal/service/mocks"
	"wordsGo/slogger"
)

func TestMain(m *testing.M) {
	slogger.MakeLogger(true)
	code := m.Run()
	os.Exit(code)
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func TestHandler_Create(t *testing.T) {

	mockUserID := uuid.New().String()
	mockTimeStr := "2025-01-01 00:00:00 +0000 UTC"
	successResponse := models.UserResponse{
		ID:        mockUserID,
		Email:     "test@example.com",
		FirstName: "John",
		LastName:  "Doe",
		CreatedAt: mockTimeStr,
		UpdatedAt: mockTimeStr,
	}

	requestBody := models.CreateUserRequest{
		Email:     "test@example.com",
		Password:  "StrongPass1!",
		FirstName: "John",
		LastName:  "Doe",
	}

	tests := []struct {
		name               string
		requestBody        any
		expectedStatusCode int
		expectedBodyJSON   any
		mockBehavior       func(s *mocks.MockUserService)
	}{
		{
			name:               "Success: User created",
			requestBody:        requestBody,
			expectedStatusCode: http.StatusCreated,
			expectedBodyJSON:   successResponse,
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("CreateUser", mock.Anything, requestBody).
					Return(&successResponse, nil).Once()
			},
		},
		{
			name:               "Failure: Duplicate Email (Conflict)",
			requestBody:        requestBody,
			expectedStatusCode: http.StatusConflict,
			expectedBodyJSON:   ErrorResponse{Error: "User already exists"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("CreateUser", mock.Anything, requestBody).
					Return(nil, models.ErrUserAlreadyExists).Once()
			},
		},
		{
			name:               "Failure: Invalid Request Body (Invalid JSON)",
			requestBody:        "this is not json", // Невалидный JSON
			expectedStatusCode: http.StatusBadRequest,
			expectedBodyJSON:   ErrorResponse{Error: "Invalid request body"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
			},
		},

		{
			name: "Failure: Missing Field (Empty Email)",
			requestBody: models.CreateUserRequest{
				Email:     "",
				Password:  "StrongPass1!",
				FirstName: "John",
				LastName:  "Doe",
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBodyJSON:   ErrorResponse{Error: "Fields cannot be empty"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.AssertNotCalled(t, "CreateUser", mock.Anything, mock.Anything)
			},
		},
		{
			name:               "Failure: Internal Service Error",
			requestBody:        requestBody,
			expectedStatusCode: http.StatusInternalServerError,
			expectedBodyJSON:   ErrorResponse{Error: "database connection failed"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("CreateUser", mock.Anything, requestBody).
					Return(nil, errors.New("database connection failed")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockUserService(t)
			tt.mockBehavior(mockService)

			handler := NewHandler(mockService)

			bodyBytes, err := json.Marshal(tt.requestBody)
			if err != nil && tt.name != "Failure: Invalid Request Body (Invalid JSON)" {
				t.Fatalf("Failed to marshal request body: %v", err)
			}

			reqBody := bytes.NewReader(bodyBytes)

			req := httptest.NewRequest(http.MethodPost, "/api/v1/users", reqBody)
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()

			handler.CreateUser(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			if tt.expectedBodyJSON != nil {
				if tt.expectedStatusCode == http.StatusCreated {
					var actualResponse models.UserResponse
					err := json.Unmarshal(rr.Body.Bytes(), &actualResponse)
					assert.NoError(t, err, "Failed to unmarshal successful response body")

					expected := tt.expectedBodyJSON.(models.UserResponse)

					assert.Equal(t, expected.ID, actualResponse.ID, "ID mismatch")
					assert.Equal(t, expected.Email, actualResponse.Email, "Email mismatch")
					assert.Equal(t, expected.FirstName, actualResponse.FirstName, "FirstName mismatch")
					assert.Equal(t, expected.LastName, actualResponse.LastName, "LastName mismatch")

					assert.NotEmpty(t, actualResponse.CreatedAt, "CreatedAt should not be empty")
					assert.NotEmpty(t, actualResponse.UpdatedAt, "UpdatedAt should not be empty")

				} else {
					expectedJSON, _ := json.Marshal(tt.expectedBodyJSON)
					assert.JSONEq(t, string(expectedJSON), rr.Body.String(), "Тело ответа ошибки не совпадает")
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_GetUserByID(t *testing.T) {
	userID := uuid.New()
	userResponse := models.UserResponse{
		ID:        userID.String(),
		Email:     "get@example.com",
		FirstName: "Get",
		LastName:  "User",
	}

	tests := []struct {
		name               string
		urlParamID         string
		expectedStatusCode int
		expectedBody       any
		mockBehavior       func(s *mocks.MockUserService)
	}{
		{
			name:               "Success: User found",
			urlParamID:         userID.String(),
			expectedStatusCode: http.StatusOK,
			expectedBody:       userResponse,
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("GetUserByID", mock.Anything, userID).
					Return(&userResponse, nil).Once()
			},
		},
		{
			name:               "Failure: Invalid UUID",
			urlParamID:         "not-a-uuid",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       ErrorResponse{Error: "Invalid user ID"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.AssertNotCalled(t, "GetUserByID", mock.Anything, mock.Anything)
			},
		},
		{
			name:               "Failure: User Not Found",
			urlParamID:         userID.String(),
			expectedStatusCode: http.StatusNotFound,
			expectedBody:       ErrorResponse{Error: "User not found"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("GetUserByID", mock.Anything, userID).
					Return(nil, models.ErrUserNotFound).Once()
			},
		},
		{
			name:               "Failure: Internal Service Error",
			urlParamID:         userID.String(),
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       ErrorResponse{Error: "Failed to get user"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("GetUserByID", mock.Anything, userID).
					Return(nil, errors.New("unexpected error")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockUserService(t)
			tt.mockBehavior(mockService)

			handler := NewHandler(mockService)

			r := chi.NewRouter()
			r.Get("/users/{id}", handler.GetUserByID)

			req := httptest.NewRequest(http.MethodGet, "/users/"+tt.urlParamID, nil)
			rr := httptest.NewRecorder()

			r.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			if tt.expectedBody != nil {
				if tt.expectedStatusCode == http.StatusOK {
					var actualResp models.UserResponse
					err := json.Unmarshal(rr.Body.Bytes(), &actualResp)
					assert.NoError(t, err)

					expected := tt.expectedBody.(models.UserResponse)
					assert.Equal(t, expected.ID, actualResp.ID)
					assert.Equal(t, expected.Email, actualResp.Email)
				} else {
					expectedJSON, _ := json.Marshal(tt.expectedBody)
					assert.JSONEq(t, string(expectedJSON), rr.Body.String())
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_GetUsers(t *testing.T) {
	usersResp := &models.ListOfUsersResponse{
		Page:  1,
		Limit: 10,
		Total: 1,
		Pages: 1,
		Data: []*models.UserResponse{
			{ID: "1", Email: "u1@example.com"},
		},
	}

	tests := []struct {
		name               string
		queryString        string
		expectedStatusCode int
		expectedBody       any
		mockBehavior       func(s *mocks.MockUserService)
	}{
		{
			name:               "Success: Default params",
			queryString:        "",
			expectedStatusCode: http.StatusOK,
			expectedBody:       usersResp,
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("GetUsers", mock.Anything, uint64(10), uint64(1), "desc").
					Return(usersResp, nil).Once()
			},
		},
		{
			name:               "Success: Custom params",
			queryString:        "?limit=5&page=2&order=asc",
			expectedStatusCode: http.StatusOK,
			expectedBody:       usersResp,
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("GetUsers", mock.Anything, uint64(5), uint64(2), "asc").
					Return(usersResp, nil).Once()
			},
		},
		{
			name:               "Failure: Invalid Limit (Not a number)",
			queryString:        "?limit=abc",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       ErrorResponse{Error: "Invalid limit"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.AssertNotCalled(t, "GetUsers", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:               "Failure: Invalid Page (Not a number)",
			queryString:        "?page=xyz",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       ErrorResponse{Error: "Invalid page"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.AssertNotCalled(t, "GetUsers", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:               "Failure: Service Error",
			queryString:        "",
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       ErrorResponse{Error: "Failed to get users"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("GetUsers", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
					Return(nil, errors.New("db error")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockUserService(t)
			tt.mockBehavior(mockService)

			handler := NewHandler(mockService)

			req := httptest.NewRequest(http.MethodGet, "/api/v1/users"+tt.queryString, nil)
			rr := httptest.NewRecorder()

			handler.GetUsers(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			if tt.expectedStatusCode == http.StatusOK {
				expectedJSON, _ := json.Marshal(tt.expectedBody)
				assert.JSONEq(t, string(expectedJSON), rr.Body.String())
			} else {
				expectedJSON, _ := json.Marshal(tt.expectedBody)
				assert.JSONEq(t, string(expectedJSON), rr.Body.String())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_Delete(t *testing.T) {
	authUserID := uuid.New()
	otherID := uuid.New()

	authUser := &models.User{
		ID:    authUserID,
		Email: "me@example.com",
	}
	tests := []struct {
		name               string
		urlParamID         string
		authUser           *models.User
		expectedStatusCode int
		expectedBody       any
		mockBehavior       func(s *mocks.MockUserService)
	}{
		{
			name:               "Success: DeleteUser own account",
			urlParamID:         authUserID.String(),
			authUser:           authUser,
			expectedStatusCode: http.StatusOK,
			expectedBody:       nil,
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("DeleteUser", mock.Anything, authUser, authUserID).
					Return(nil).Once()
			},
		},
		{
			name:               "Failure: User not found",
			urlParamID:         otherID.String(),
			authUser:           authUser,
			expectedStatusCode: http.StatusNotFound,
			expectedBody:       ErrorResponse{Error: "User not found"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("DeleteUser", mock.Anything, authUser, otherID).Return(models.ErrUserNotFound).Once()
			},
		},
		{
			name:               "Failure: Permission denied",
			urlParamID:         otherID.String(),
			authUser:           authUser,
			expectedStatusCode: http.StatusForbidden,
			expectedBody:       ErrorResponse{Error: "Permission denied"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("DeleteUser", mock.Anything, authUser, otherID).Return(models.ErrPermissionDenied).Once()
			},
		},
		{
			name:               "Failure: Invalid UUID",
			urlParamID:         "invalid-uuid-string",
			authUser:           authUser,
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       ErrorResponse{Error: "Invalid user ID"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.AssertNotCalled(t, "DeleteUser", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:               "Failure: Internal error",
			urlParamID:         authUserID.String(),
			authUser:           authUser,
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       ErrorResponse{Error: "Failed to delete user"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("DeleteUser", mock.Anything, authUser, authUserID).
					Return(errors.New("db connection lost")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockUserService(t)
			tt.mockBehavior(mockService)

			handler := NewHandler(mockService)

			r := chi.NewRouter()
			r.Delete("/users/{id}", handler.DeleteUser)

			req := httptest.NewRequest(http.MethodDelete, "/users/"+tt.urlParamID, nil)

			if tt.authUser != nil {
				ctx := context.WithValue(req.Context(), middleware.UserCtxKey{}, tt.authUser)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			if tt.expectedBody != nil {
				expectedJSON, _ := json.Marshal(tt.expectedBody)
				assert.JSONEq(t, string(expectedJSON), rr.Body.String())
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_Update(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	myID := uuid.New()
	otherID := uuid.New()

	authUser := &models.User{
		ID:    myID,
		Email: "me@example.com",
	}

	updatedResponse := models.UserResponse{
		ID:        myID.String(),
		Email:     "new@email.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	tests := []struct {
		name               string
		urlParamID         string
		authUser           *models.User
		requestBody        any
		expectedStatusCode int
		expectedBody       any
		mockBehavior       func(s *mocks.MockUserService)
	}{
		{
			name:       "Success: UpdateUser self (Email only)",
			urlParamID: myID.String(),
			authUser:   authUser,
			requestBody: models.UpdateUserRequest{
				Email: strPtr("new@email.com"),
			},
			expectedStatusCode: http.StatusOK,
			expectedBody:       updatedResponse,
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("UpdateUser", mock.Anything, myID, models.UpdateUserRequest{
					Email: strPtr("new@email.com"),
				}).Return(&updatedResponse, nil).Once()
			},
		},
		{
			name:       "Failure: UpdateUser other user (Forbidden)",
			urlParamID: otherID.String(),
			authUser:   authUser,
			requestBody: models.UpdateUserRequest{
				Email: strPtr("hacker@email.com"),
			},
			expectedStatusCode: http.StatusForbidden,
			expectedBody:       ErrorResponse{Error: "You can update only your own account"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:       "Failure: Invalid Password (Validation)",
			urlParamID: myID.String(),
			authUser:   authUser,
			requestBody: models.UpdateUserRequest{
				Password: strPtr("short"),
			},
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       ErrorResponse{Error: "password must be at least 8 characters long"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:               "Failure: Invalid JSON Body",
			urlParamID:         myID.String(),
			authUser:           authUser,
			requestBody:        "invalid-json",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       ErrorResponse{Error: "Invalid request body"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.AssertNotCalled(t, "UpdateUser", mock.Anything, mock.Anything, mock.Anything)
			},
		},
		{
			name:       "Failure: Service Error",
			urlParamID: myID.String(),
			authUser:   authUser,
			requestBody: models.UpdateUserRequest{
				FirstName: strPtr("NewName"),
			},
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       ErrorResponse{Error: "Failed to update user"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("UpdateUser", mock.Anything, myID, mock.Anything).
					Return(nil, errors.New("db error")).Once()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockUserService(t)
			tt.mockBehavior(mockService)

			handler := NewHandler(mockService)

			r := chi.NewRouter()
			r.Put("/users/{id}", handler.UpdateUser)

			var reqBody *bytes.Reader
			if s, ok := tt.requestBody.(string); ok {
				reqBody = bytes.NewReader([]byte(s))
			} else {
				bodyBytes, _ := json.Marshal(tt.requestBody)
				reqBody = bytes.NewReader(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPut, "/users/"+tt.urlParamID, reqBody)

			if tt.authUser != nil {
				ctx := context.WithValue(req.Context(), middleware.UserCtxKey{}, tt.authUser)
				req = req.WithContext(ctx)
			}

			rr := httptest.NewRecorder()
			r.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			if tt.expectedBody != nil {
				if tt.expectedStatusCode == http.StatusOK {
					var actualResp models.UserResponse
					_ = json.Unmarshal(rr.Body.Bytes(), &actualResp)

					expected := tt.expectedBody.(models.UserResponse)
					assert.Equal(t, expected.ID, actualResp.ID)
					assert.Equal(t, expected.Email, actualResp.Email)
				} else {
					expectedJSON, _ := json.Marshal(tt.expectedBody)
					assert.JSONEq(t, string(expectedJSON), rr.Body.String())
				}
			}

			mockService.AssertExpectations(t)
		})
	}
}

func TestHandler_Login(t *testing.T) {
	request := models.LoginRequest{
		Email:    "test@test.com",
		Password: "password",
	}
	token := "token"
	tests := []struct {
		name               string
		requestBody        any
		expectedStatusCode int
		expectedBody       any
		mockBehavior       func(s *mocks.MockUserService)
	}{
		{
			name:               "Success: Valid credentials",
			requestBody:        request,
			expectedStatusCode: http.StatusOK,
			expectedBody:       models.LoginResponse{Token: token},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("Login", mock.Anything, request).Return(token, nil).Once()
			},
		},
		{
			name:               "Failure: Invalid credentials",
			requestBody:        request,
			expectedStatusCode: http.StatusUnauthorized,
			expectedBody:       ErrorResponse{Error: "Invalid email or password"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("Login", mock.Anything, request).Return("", models.ErrInvalidCredentials).Once()
			},
		},
		{
			name:               "Failure: Internal Server Error",
			requestBody:        request,
			expectedStatusCode: http.StatusInternalServerError,
			expectedBody:       ErrorResponse{Error: "Internal server error"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.On("Login", mock.Anything, request).Return("", errors.New("unexpected error")).Once()
			},
		},
		{
			name:               "Failure: Invalid JSON Body",
			requestBody:        "invalid-json",
			expectedStatusCode: http.StatusBadRequest,
			expectedBody:       ErrorResponse{Error: "Invalid request body"},
			mockBehavior: func(s *mocks.MockUserService) {
				s.AssertNotCalled(t, "Login", mock.Anything, mock.Anything)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := mocks.NewMockUserService(t)
			tt.mockBehavior(mockService)

			handler := NewHandler(mockService)

			var reqBodyReader *bytes.Reader
			if s, ok := tt.requestBody.(string); ok {
				reqBodyReader = bytes.NewReader([]byte(s))
			} else {
				bodyBytes, _ := json.Marshal(tt.requestBody)
				reqBodyReader = bytes.NewReader(bodyBytes)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/login", reqBodyReader)
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			handler.Login(rr, req)

			assert.Equal(t, tt.expectedStatusCode, rr.Code)

			if tt.expectedBody != nil {
				expectedJSON, _ := json.Marshal(tt.expectedBody)
				assert.JSONEq(t, string(expectedJSON), rr.Body.String())
			}

			mockService.AssertExpectations(t)
		})
	}
}
