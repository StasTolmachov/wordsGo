package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"wordsGo/internal/models"
	"wordsGo/internal/utils"
)

func TestAuthMiddleware_MissingAuthHeader(t *testing.T) {
	secret := "test-secret"
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := AuthMidleware(secret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, called)
}

func TestAuthMiddleware_InvalidAuthHeader(t *testing.T) {
	secret := "test-secret"
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := AuthMidleware(secret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Token abc")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, called)
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	secret := "test-secret"
	called := false

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := AuthMidleware(secret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, called)
}

func TestAuthMiddleware_InvalidUserID(t *testing.T) {
	secret := "test-secret"
	called := false

	claims := utils.CustomClaims{
		UserID: "not-a-uuid",
		Role:   "user",
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(secret))
	assert.NoError(t, err)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	})

	handler := AuthMidleware(secret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
	assert.False(t, called)
}

func TestAuthMiddleware_Success(t *testing.T) {
	secret := "test-secret"
	called := false
	userID := uuid.New()
	role := "admin"

	tokenString, err := utils.GenerateToken(userID, role, secret, time.Hour)
	assert.NoError(t, err)

	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		user, ok := r.Context().Value(UserCtxKey{}).(*models.User)
		assert.True(t, ok)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, models.UserRole(role), user.Role)
		w.WriteHeader(http.StatusOK)
	})

	handler := AuthMidleware(secret)(next)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Authorization", "Bearer "+tokenString)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, called)
}
