package utils

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestJWT(t *testing.T) {
	secret := "super-secret-key"
	userID := uuid.New()
	role := "admin"
	ttl := time.Hour

	t.Run("Generate and Parse Token", func(t *testing.T) {
		tokenString, err := GenerateToken(userID, role, secret, ttl)
		assert.NoError(t, err)
		assert.NotEmpty(t, tokenString)

		claims, err := ParseToken(tokenString, secret)
		assert.NoError(t, err)
		assert.NotNil(t, claims)

		assert.Equal(t, userID.String(), claims.UserID)
		assert.Equal(t, role, claims.Role)
	})

	t.Run("Parse Invalid Token (Wrong Secret)", func(t *testing.T) {
		tokenString, _ := GenerateToken(userID, role, secret, ttl)

		_, err := ParseToken(tokenString, "wrong-secret")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "signature is invalid")
	})

	t.Run("Parse Expired Token", func(t *testing.T) {
		tokenString, _ := GenerateToken(userID, role, secret, -time.Second)

		_, err := ParseToken(tokenString, secret)
		assert.Error(t, err)
	})
}
