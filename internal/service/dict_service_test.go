package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"wordsGo/internal/models"
	"wordsGo/internal/repository/modelsDB"
)

// MockDictionaryRepo is a mock implementation of repository.DictionaryRepository
type MockDictionaryRepo struct {
	mock.Mock
}

func (m *MockDictionaryRepo) DictionaryInsert(ctx context.Context, dictionary []modelsDB.DictionaryDB) error {
	args := m.Called(ctx, dictionary)
	return args.Error(0)
}

func (m *MockDictionaryRepo) GetWords(ctx context.Context, userID uuid.UUID, filter, sortBy, order string, pagination modelsDB.Pagination) ([]modelsDB.UserWordDB, uint64, error) {
	args := m.Called(ctx, userID, filter, sortBy, order, pagination)
	return args.Get(0).([]modelsDB.UserWordDB), args.Get(1).(uint64), args.Error(2)
}

func (m *MockDictionaryRepo) SearchByOriginal(ctx context.Context, query string) ([]modelsDB.DictionaryDB, error) {
	args := m.Called(ctx, query)
	return args.Get(0).([]modelsDB.DictionaryDB), args.Error(1)
}

func (m *MockDictionaryRepo) AddWordToUser(ctx context.Context, userID, wordID uuid.UUID) error {
	args := m.Called(ctx, userID, wordID)
	return args.Error(0)
}

func (m *MockDictionaryRepo) GetLessonWords(ctx context.Context, userID uuid.UUID) ([]modelsDB.LessonWordDB, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).([]modelsDB.LessonWordDB), args.Error(1)
}

func (m *MockDictionaryRepo) GetRandomWords(ctx context.Context, userID uuid.UUID, limit int, excludeIDs []uuid.UUID) ([]modelsDB.LessonWordDB, error) {
	args := m.Called(ctx, userID, limit, excludeIDs)
	return args.Get(0).([]modelsDB.LessonWordDB), args.Error(1)
}

func (m *MockDictionaryRepo) GetUserProgress(ctx context.Context, userID, wordID uuid.UUID) (*modelsDB.UserProgressDB, error) {
	args := m.Called(ctx, userID, wordID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*modelsDB.UserProgressDB), args.Error(1)
}

func (m *MockDictionaryRepo) SaveUserProgress(ctx context.Context, p *modelsDB.UserProgressDB) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func (m *MockDictionaryRepo) DeleteUserProgress(ctx context.Context, userID, wordID uuid.UUID) error {
	args := m.Called(ctx, userID, wordID)
	return args.Error(0)
}

func (m *MockDictionaryRepo) DeleteAllUserProgress(ctx context.Context, userID uuid.UUID) error {
	args := m.Called(ctx, userID)
	return args.Error(0)
}

func (m *MockDictionaryRepo) AddWordsByLevel(ctx context.Context, userID uuid.UUID, level string) (int64, error) {
	args := m.Called(ctx, userID, level)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDictionaryRepo) GetProgressStats(ctx context.Context, userID uuid.UUID) (map[string]float64, float64, error) {
	args := m.Called(ctx, userID)
	return args.Get(0).(map[string]float64), args.Get(1).(float64), args.Error(2)
}

func (m *MockDictionaryRepo) UpdateUserWordDetails(ctx context.Context, p *modelsDB.UserProgressDB) error {
	args := m.Called(ctx, p)
	return args.Error(0)
}

func TestProcessAnswer_Correct(t *testing.T) {
	mockRepo := new(MockDictionaryRepo)
	service := NewDictionaryService(mockRepo)

	userID := uuid.New()
	wordID := uuid.New()
	wordIDStr := wordID.String()

	initialProgress := &modelsDB.UserProgressDB{
		UserID:          userID,
		WordID:          wordID,
		TotalMistakes:   5,
		CorrectStreak:   1,
		DifficultyLevel: 0.5,
	}

	mockRepo.On("GetUserProgress", mock.Anything, userID, wordID).Return(initialProgress, nil)

	// Expect SaveUserProgress with TotalMistakes = 0 (delta)
	mockRepo.On("SaveUserProgress", mock.Anything, mock.MatchedBy(func(p *modelsDB.UserProgressDB) bool {
		return p.TotalMistakes == 0 && p.CorrectStreak == 2
	})).Return(nil)

	req := models.SubmitAnswerRequest{
		WordID:    wordIDStr,
		IsCorrect: true,
	}

	resp, err := service.ProcessAnswer(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.Equal(t, 2, resp.CorrectStreak)
	mockRepo.AssertExpectations(t)
}

func TestProcessAnswer_Incorrect(t *testing.T) {
	mockRepo := new(MockDictionaryRepo)
	service := NewDictionaryService(mockRepo)

	userID := uuid.New()
	wordID := uuid.New()
	wordIDStr := wordID.String()

	initialProgress := &modelsDB.UserProgressDB{
		UserID:          userID,
		WordID:          wordID,
		TotalMistakes:   5,
		CorrectStreak:   2,
		DifficultyLevel: 0.5,
	}

	mockRepo.On("GetUserProgress", mock.Anything, userID, wordID).Return(initialProgress, nil)

	// Expect SaveUserProgress with TotalMistakes = 1 (delta)
	mockRepo.On("SaveUserProgress", mock.Anything, mock.MatchedBy(func(p *modelsDB.UserProgressDB) bool {
		return p.TotalMistakes == 1 && p.CorrectStreak == 0
	})).Return(nil)

	req := models.SubmitAnswerRequest{
		WordID:    wordIDStr,
		IsCorrect: false,
	}

	resp, err := service.ProcessAnswer(context.Background(), userID, req)

	assert.NoError(t, err)
	assert.Equal(t, 0, resp.CorrectStreak)
	mockRepo.AssertExpectations(t)
}
