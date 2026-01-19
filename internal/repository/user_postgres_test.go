//go:build integration

package repository

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	"wordsGo/internal/config"
	"wordsGo/internal/repository/modelsDB"
)

type UserRepoSuite struct {
	suite.Suite
	dbContainer *postgres.PostgresContainer
	pg          *Postgres
	repo        *UserRepo
}

func (s *UserRepoSuite) SetupSuite() {
	ctx := context.Background()

	dbContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("user_db_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)))

	if err != nil {
		s.T().Fatal(err)
	}
	s.dbContainer = dbContainer

	host, err := dbContainer.Host(ctx)
	s.Require().NoError(err)
	port, err := dbContainer.MappedPort(ctx, "5432")
	s.Require().NoError(err)

	cfg := config.DB{
		Host:        host,
		Port:        port.Port(),
		Username:    "postgres",
		Password:    "password",
		Database:    "user_db_test",
		SSLMode:     "disable",
		MigratePath: "file://../../migrations",
	}

	s.pg, err = NewPostgres(cfg)
	if err != nil {
		s.T().Fatal(err)
	}
	s.repo = NewUserRepo(s.pg)
}

func (s *UserRepoSuite) TearDownSuite() {
	if s.dbContainer != nil {
		s.dbContainer.Terminate(context.Background())
	}
	if s.pg != nil {
		s.pg.Close()
	}
}

func (s *UserRepoSuite) SetupTest() {
	_, err := s.pg.db.Exec("TRUNCATE TABLE users CASCADE")
	s.Require().NoError(err)
}

func TestUserRepoSuite(t *testing.T) {
	suite.Run(t, new(UserRepoSuite))
}

func (s *UserRepoSuite) TestCreateUser() {
	newUser := &modelsDB.UserDB{
		Email:        "integration@test.com",
		PasswordHash: "someHash",
		FirstName:    "Test",
		LastName:     "User",
		Role:         "user",
	}

	s.Run("Success create user", func() {
		createdUser, err := s.repo.Create(context.Background(), newUser)

		s.Require().NoError(err)
		s.NotNil(createdUser)
		s.NotEmpty(createdUser.ID)
		s.Equal(newUser.Email, createdUser.Email)
		s.WithinDuration(time.Now(), createdUser.CreatedAt, 2*time.Second)

		var getUser modelsDB.UserDB
		s.pg.db.QueryRow("select id, email from users where id=$1", createdUser.ID).Scan(&getUser.ID, &getUser.Email)
		s.Require().NoError(err)
		s.Equal(createdUser.ID, getUser.ID)
		s.Equal(newUser.Email, getUser.Email)
	})

	s.Run("Error_Duplicate_Email", func() {
		duplicateUser, err := s.repo.Create(context.Background(), newUser)
		s.Require().Error(err, modelsDB.ErrDuplicateEmail)
		s.Nil(duplicateUser)
	})
}

func (s *UserRepoSuite) TestGetPasswordHashByEmail() {

	newUser := &modelsDB.UserDB{
		Email:        "integration@test.com",
		PasswordHash: "someHash",
		FirstName:    "Test",
		LastName:     "User",
	}
	s.Run("User not found", func() {
		gotUser, err := s.repo.GetPasswordHashByEmail(context.Background(), newUser.Email)
		s.Require().Error(err, modelsDB.ErrUserNotFound)
		s.Nil(gotUser)
	})

	s.Run("Success get password", func() {
		query := `
		insert into users 
    	(email, password_hash, first_name, last_name)
		values ($1, $2, $3, $4)
		`

		s.pg.db.QueryRowxContext(context.Background(), query,
			newUser.Email,
			newUser.PasswordHash,
			newUser.FirstName,
			newUser.LastName,
		)

		gotUser, _ := s.repo.GetPasswordHashByEmail(context.Background(), newUser.Email)
		s.Require().Equal(newUser.PasswordHash, gotUser.PasswordHash)
	})
}

func (s *UserRepoSuite) TestUpdateUser() {
	userToUpdate := &modelsDB.UserDB{
		Email:        "update_test@example.com",
		PasswordHash: "old_hash",
		FirstName:    "OldName",
		LastName:     "OldLast",
		Role:         "user",
	}
	createdUser, err := s.repo.Create(context.Background(), userToUpdate)
	s.Require().NoError(err)

	s.Run("Success: Update fields", func() {
		time.Sleep(time.Millisecond * 100)

		newEmail := "updated_email@example.com"
		newName := "NewName"

		fields := map[string]any{
			"email":      newEmail,
			"first_name": newName,
		}

		updatedUser, err := s.repo.Update(context.Background(), createdUser.ID, fields)
		s.Require().NoError(err)
		s.NotNil(updatedUser)

		s.Equal(newEmail, updatedUser.Email)
		s.Equal(newName, updatedUser.FirstName)
		s.Equal(createdUser.LastName, updatedUser.LastName)
		s.True(updatedUser.UpdatedAt.After(createdUser.UpdatedAt), "UpdatedAt should be updated")

		dbUser, err := s.repo.GetUserByID(context.Background(), createdUser.ID)
		s.Require().NoError(err)
		s.Equal(newEmail, dbUser.Email)
	})

	s.Run("Failure: Update forbidden column", func() {
		fields := map[string]any{
			"id": "some-new-id",
		}
		_, err := s.repo.Update(context.Background(), createdUser.ID, fields)
		s.Error(err)
		s.Contains(err.Error(), "column id is not allowed")
	})

	s.Run("Failure: User not found", func() {
		fields := map[string]any{
			"first_name": "Ghost",
		}
		id, err := uuid.NewUUID()
		s.Require().NoError(err)

		_, err = s.repo.Update(context.Background(), id, fields)
		fmt.Println(err)
		s.Require().Error(err, modelsDB.ErrUserNotFound)
	})
}

func (s *UserRepoSuite) TestGetUsers() {
	emails := []string{"a@test.com", "b@test.com", "c@test.com"}
	for _, email := range emails {
		_, err := s.repo.Create(context.Background(), &modelsDB.UserDB{
			Email:        email,
			PasswordHash: "hash",
			FirstName:    "Test",
			LastName:     "User",
			Role:         "user",
		})
		s.Require().NoError(err)
		time.Sleep(time.Millisecond * 10)
	}

	s.Run("Success: Pagination Limit & Offset", func() {
		pagination := modelsDB.Pagination{Limit: 2, Offset: 0}
		users, total, err := s.repo.GetUsers(context.Background(), "ASC", pagination)

		s.Require().NoError(err)
		s.Equal(uint64(3), total)
		s.Len(users, 2)
		s.Equal("a@test.com", users[0].Email)
		s.Equal("b@test.com", users[1].Email)

		pagination = modelsDB.Pagination{Limit: 2, Offset: 2}
		users, total, err = s.repo.GetUsers(context.Background(), "ASC", pagination)

		s.Require().NoError(err)
		s.Equal(uint64(3), total)
		s.Len(users, 1)
		s.Equal("c@test.com", users[0].Email)
	})

	s.Run("Success: Sorting DESC", func() {
		pagination := modelsDB.Pagination{Limit: 10, Offset: 0}
		users, _, err := s.repo.GetUsers(context.Background(), "DESC", pagination)

		s.Require().NoError(err)
		s.Len(users, 3)

		s.Equal("c@test.com", users[0].Email)
		s.Equal("a@test.com", users[len(users)-1].Email)
	})

}

func (s *UserRepoSuite) TestDelete() {
	newUser := &modelsDB.UserDB{
		Email:        "integration@test.com",
		PasswordHash: "someHash",
		FirstName:    "Test",
		LastName:     "User",
		Role:         "user",
	}
	createdUser, err := s.repo.Create(context.Background(), newUser)
	s.Require().NoError(err)

	s.Run("Success delete user", func() {
		err := s.repo.Delete(context.Background(), createdUser.ID)
		s.Require().NoError(err)
		_, err = s.repo.GetUserByID(context.Background(), createdUser.ID)
		s.Require().Error(err, modelsDB.ErrUserNotFound)
	})

}
