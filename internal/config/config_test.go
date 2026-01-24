package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"wordsGo/slogger"
)

func TestLoad(t *testing.T) {
	slogger.MakeLogger(true)

	t.Run("Success: Load valid config", func(t *testing.T) {

		t.Setenv("DB_HOST", "localhost")
		t.Setenv("DB_PORT", "5432")
		t.Setenv("DB_USER", "postgres")
		t.Setenv("DB_PASSWORD", "secret")
		t.Setenv("DB_NAME", "users_db")
		t.Setenv("DB_MIGRATE_PATH", "file://migrations")
		t.Setenv("API_PORT", "8080")
		t.Setenv("JWT_SECRET", "secret")
		t.Setenv("ADMIN_EMAIL", "admin@example.com")
		t.Setenv("ADMIN_PASSWORD", "password")

		cfg, err := Load()
		assert.NoError(t, err)
		assert.NotNil(t, cfg)
		assert.Equal(t, "localhost", cfg.DB.Host)
		assert.Equal(t, "disable", cfg.DB.SSLMode)
		assert.Equal(t, "8080", cfg.Api.Port)
	})

	//t.Run("Failure: Missing required env var", func(t *testing.T) {
	//
	//	cfg, err := Load()
	//	assert.Error(t, err)
	//	assert.Nil(t, cfg)
	//	assert.Contains(t, err.Error(), "environment variable not set")
	//})
}

func TestDB_DSN(t *testing.T) {
	dbCfg := DB{
		Host:     "localhost",
		Port:     "5432",
		Username: "user",
		Password: "pass",
		Database: "mydb",
		SSLMode:  "disable",
	}

	expectedDSN := "postgres://user:pass@localhost:5432/mydb?sslmode=disable"
	assert.Equal(t, expectedDSN, dbCfg.DSN())
}
