package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"

	"wordsGo/internal/models"
	"wordsGo/internal/repository"
	"wordsGo/internal/repository/modelsDB"
	"wordsGo/slogger"
)

type DictionaryService interface {
	LoadDictionary(ctx context.Context, path string) error
	GetWords(ctx context.Context, userID uuid.UUID, limit, page uint64, order string) (*models.ListOfWordsResponse, error)
	AddWords(ctx context.Context, word models.DictionaryWord) error
	SearchWords(ctx context.Context, query string) ([]*models.DictionaryWord, error)
	AddWordToLearning(ctx context.Context, userID uuid.UUID, wordIDStr string) error
}

type dictionaryService struct {
	repo *repository.DictionaryRepo
}

func NewDictionaryService(repo *repository.DictionaryRepo) DictionaryService {
	return &dictionaryService{
		repo: repo,
	}
}

func (s *dictionaryService) LoadDictionary(ctx context.Context, path string) error {
	// 1. Проверяем наличие файла
	if _, err := os.Stat(path); os.IsNotExist(err) {
		slogger.Log.WarnContext(ctx, "Dictionary file not found, skipping", "path", path)
		return nil
	}

	// 2. Читаем файл
	byteValue, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// 3. Парсим JSON
	var words []models.DictionaryWord
	if err := json.Unmarshal(byteValue, &words); err != nil {
		return fmt.Errorf("failed to unmarshal json: %w", err)
	}

	if len(words) == 0 {
		return nil
	}

	// 4. Конвертируем в DB-модели
	dbWords := make([]modelsDB.DictionaryDB, 0, len(words))
	for _, w := range words {
		dbWords = append(dbWords, models.ToDictionaryDB(w))
	}

	// 5. Сохраняем в БД через репозиторий
	if err := s.repo.DictionaryInsert(ctx, dbWords); err != nil {
		return fmt.Errorf("service failed to insert dictionary: %w", err)
	}

	slogger.Log.InfoContext(ctx, "Dictionary loaded successfully", "count", len(dbWords))
	return nil
}

func (s *dictionaryService) GetWords(ctx context.Context, userID uuid.UUID, limit, page uint64, order string) (*models.ListOfWordsResponse, error) {

	if limit == 0 {
		limit = 10
	}
	if page == 0 {
		page = 1
	}

	offset := (page - 1) * limit
	pagination := &modelsDB.Pagination{
		Limit:  limit,
		Offset: offset,
	}
	wordsDB, total, err := s.repo.GetWords(ctx, userID, order, *pagination)
	if err != nil {
		return nil, err
	}

	wordsResponse := make([]*models.DictionaryWord, len(wordsDB))
	for i, wordModel := range wordsDB {
		wordsResponse[i] = models.FromDictionaryDB(&wordModel)
	}

	pages := (total + limit - 1) / limit

	resp := &models.ListOfWordsResponse{
		Page:  page,
		Limit: limit,
		Total: total,
		Pages: pages,
		Data:  wordsResponse,
	}

	return resp, nil
}

func (s *dictionaryService) AddWords(ctx context.Context, word models.DictionaryWord) error {
	return nil

}

func (s *dictionaryService) SearchWords(ctx context.Context, query string) ([]*models.DictionaryWord, error) {
	wordsDB, err := s.repo.SearchByOriginal(ctx, query)
	if err != nil {
		return nil, err
	}

	// Конвертация из модели БД в JSON-модель
	result := make([]*models.DictionaryWord, len(wordsDB))
	for i, w := range wordsDB {
		result[i] = models.FromDictionaryDB(&w)
	}
	return result, nil
}
func (s *dictionaryService) AddWordToLearning(ctx context.Context, userID uuid.UUID, wordIDStr string) error {
	wordID, err := uuid.Parse(wordIDStr)
	if err != nil {
		return fmt.Errorf("invalid word id: %w", err)
	}

	// Тут можно добавить проверку, существует ли слово в словаре, если нет внешних ключей (но у вас они есть).

	err = s.repo.AddWordToUser(ctx, userID, wordID)
	if err != nil {
		// Тут можно обработать ошибку дубликата (код 23505 в Postgres), если нужно сообщить юзеру "уже добавлено"
		return fmt.Errorf("failed to add word to user: %w", err)
	}
	return nil
}
