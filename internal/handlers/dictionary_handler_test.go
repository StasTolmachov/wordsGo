package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"wordsGo/internal/config"
	"wordsGo/internal/middleware"
	"wordsGo/internal/models"
)

type stubUserService struct{}

func (s stubUserService) Create(ctx context.Context, req models.CreateUserRequest) (*models.UserResponse, error) {
	return nil, nil
}
func (s stubUserService) Authenticate(ctx context.Context, email, password string) (*models.User, error) {
	return nil, nil
}
func (s stubUserService) GetUserByID(ctx context.Context, id uuid.UUID) (*models.UserResponse, error) {
	return nil, nil
}
func (s stubUserService) Delete(ctx context.Context, requester *models.User, targetID uuid.UUID) error {
	return nil
}
func (s stubUserService) Update(ctx context.Context, id uuid.UUID, req models.UpdateUserRequest) (*models.UserResponse, error) {
	return nil, nil
}
func (s stubUserService) GetUsers(ctx context.Context, limit, page uint64, order string) (*models.ListOfUsersResponse, error) {
	return nil, nil
}
func (s stubUserService) Login(ctx context.Context, req models.LoginRequest) (string, error) {
	return "", nil
}
func (s stubUserService) SyncAdmin(ctx context.Context, adminCfg config.Admin) error {
	return nil
}

type stubDictionaryService struct {
	searchCalled bool
	searchQuery  string
	searchResult []*models.DictionaryWord
	searchErr    error

	addUserID uuid.UUID
	addWordID string
	addErr    error
	addCalled bool

	lessonUserID uuid.UUID
	lessonResp   *models.LessonResponse
	lessonErr    error
	lessonCalled bool

	answerUserID uuid.UUID
	answerReq    models.SubmitAnswerRequest
	answerResp   *models.SubmitAnswerResponse
	answerErr    error
	answerCalled bool
}

func (s *stubDictionaryService) LoadDictionary(ctx context.Context, path, langCode string) error {
	return nil
}
func (s *stubDictionaryService) GetWords(ctx context.Context, userID uuid.UUID, langCode, filter string, limit, page uint64, order string) (*models.ListOfWordsResponse, error) {
	return nil, nil
}
func (s *stubDictionaryService) SearchWords(ctx context.Context, query, langCode string) ([]*models.DictionaryWord, error) {
	s.searchCalled = true
	s.searchQuery = query
	return s.searchResult, s.searchErr
}
func (s *stubDictionaryService) AddWordToLearning(ctx context.Context, userID uuid.UUID, wordIDStr, langCode string) error {
	s.addCalled = true
	s.addUserID = userID
	s.addWordID = wordIDStr
	return s.addErr
}
func (s *stubDictionaryService) GenerateLesson(ctx context.Context, userID uuid.UUID, langCode string) (*models.LessonResponse, error) {
	s.lessonCalled = true
	s.lessonUserID = userID
	return s.lessonResp, s.lessonErr
}
func (s *stubDictionaryService) ProcessAnswer(ctx context.Context, userID uuid.UUID, req models.SubmitAnswerRequest, langCode string) (*models.SubmitAnswerResponse, error) {
	s.answerCalled = true
	s.answerUserID = userID
	s.answerReq = req
	return s.answerResp, s.answerErr
}

func (s *stubDictionaryService) DeleteWordFromLearning(ctx context.Context, userID uuid.UUID, wordIDStr string) error {
	// Simple stub implementation
	return nil
}

func (s *stubDictionaryService) AddWordsByLevel(ctx context.Context, userID uuid.UUID, level, langCode string) (int64, error) {
	return 0, nil
}

func (s *stubDictionaryService) MarkAsLearned(ctx context.Context, userID uuid.UUID, wordIDStr string) error {
	return nil
}

func (s *stubDictionaryService) UpdateWordDetails(ctx context.Context, userID uuid.UUID, wordIDStr string, req models.UpdateWordRequest) error {
	return nil
}

func (s *stubDictionaryService) ResetProgress(ctx context.Context, userID uuid.UUID) error {
	return nil
}

func TestHandler_SearchDictionary_EmptyQuery(t *testing.T) {
	dict := &stubDictionaryService{}
	handler := NewHandler(stubUserService{}, dict)

	req := httptest.NewRequest(http.MethodGet, "/dictionary/search", nil)
	userID := uuid.New()
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserCtxKey{}, &models.User{ID: userID, TargetLang: "en"}))
	rr := httptest.NewRecorder()

	handler.SearchDictionary(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.False(t, dict.searchCalled)

	var words []models.DictionaryWord
	err := json.Unmarshal(rr.Body.Bytes(), &words)
	assert.NoError(t, err)
	assert.Len(t, words, 0)
}

func TestHandler_SearchDictionary_Success(t *testing.T) {
	dict := &stubDictionaryService{
		searchResult: []*models.DictionaryWord{
			{Original: "apple", Translation: "yabloko"},
			{Original: "apply", Translation: "primenyat"},
		},
	}
	handler := NewHandler(stubUserService{}, dict)

	req := httptest.NewRequest(http.MethodGet, "/dictionary/search?q=app", nil)
	userID := uuid.New()
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserCtxKey{}, &models.User{ID: userID, TargetLang: "en"}))
	rr := httptest.NewRecorder()

	handler.SearchDictionary(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, dict.searchCalled)
	assert.Equal(t, "app", dict.searchQuery)

	var words []models.DictionaryWord
	err := json.Unmarshal(rr.Body.Bytes(), &words)
	assert.NoError(t, err)
	assert.Len(t, words, 2)
	assert.Equal(t, "apple", words[0].Original)
}

func TestHandler_AddWordToLearning_Unauthorized(t *testing.T) {
	dict := &stubDictionaryService{}
	handler := NewHandler(stubUserService{}, dict)

	req := httptest.NewRequest(http.MethodPost, "/users/words", bytes.NewReader([]byte(`{"word_id":"abc"}`)))
	rr := httptest.NewRecorder()

	handler.AddWordToLearning(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
	assert.False(t, dict.addCalled)
}

func TestHandler_AddWordToLearning_Success(t *testing.T) {
	dict := &stubDictionaryService{}
	handler := NewHandler(stubUserService{}, dict)

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/users/words", bytes.NewReader([]byte(`{"word_id":"word-123"}`)))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserCtxKey{}, &models.User{ID: userID}))
	rr := httptest.NewRecorder()

	handler.AddWordToLearning(rr, req)

	assert.Equal(t, http.StatusCreated, rr.Code)
	assert.True(t, dict.addCalled)
	assert.Equal(t, userID, dict.addUserID)
	assert.Equal(t, "word-123", dict.addWordID)
}

func TestHandler_StartLesson_Success(t *testing.T) {
	dict := &stubDictionaryService{
		lessonResp: &models.LessonResponse{
			Words: []models.WordResponse{
				{ID: uuid.New().String(), Original: "apple", Translation: "yabloko"},
			},
		},
	}
	handler := NewHandler(stubUserService{}, dict)

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/lesson/start", nil)
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserCtxKey{}, &models.User{ID: userID}))
	rr := httptest.NewRecorder()

	handler.StartLesson(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, dict.lessonCalled)
	assert.Equal(t, userID, dict.lessonUserID)

	var resp models.LessonResponse
	err := json.Unmarshal(rr.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Len(t, resp.Words, 1)
	assert.Equal(t, "apple", resp.Words[0].Original)
}

func TestHandler_SubmitAnswer_BadRequest(t *testing.T) {
	dict := &stubDictionaryService{}
	handler := NewHandler(stubUserService{}, dict)

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/lesson/answer", bytes.NewReader([]byte(`{"word_id":""}`)))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserCtxKey{}, &models.User{ID: userID}))
	rr := httptest.NewRecorder()

	handler.SubmitAnswer(rr, req)

	assert.Equal(t, http.StatusBadRequest, rr.Code)
	assert.False(t, dict.answerCalled)
}

func TestHandler_SubmitAnswer_Success(t *testing.T) {
	dict := &stubDictionaryService{
		answerResp: &models.SubmitAnswerResponse{
			WordID:        "word-123",
			NewDifficulty: 0.2,
			IsLearned:     false,
			CorrectStreak: 1,
		},
	}
	handler := NewHandler(stubUserService{}, dict)

	userID := uuid.New()
	req := httptest.NewRequest(http.MethodPost, "/lesson/answer", bytes.NewReader([]byte(`{"word_id":"word-123","is_correct":true}`)))
	req = req.WithContext(context.WithValue(req.Context(), middleware.UserCtxKey{}, &models.User{ID: userID}))
	rr := httptest.NewRecorder()

	handler.SubmitAnswer(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.True(t, dict.answerCalled)
	assert.Equal(t, userID, dict.answerUserID)
	assert.Equal(t, "word-123", dict.answerReq.WordID)
	assert.True(t, dict.answerReq.IsCorrect)
}
