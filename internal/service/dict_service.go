package service

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"wordsGo/internal/models"
	"wordsGo/internal/repository"
	"wordsGo/internal/repository/modelsDB"
	"wordsGo/slogger"
)

type DictionaryService interface {
	LoadDictionary(ctx context.Context, path string) error
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
