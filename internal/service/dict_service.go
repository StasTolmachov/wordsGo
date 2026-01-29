package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/google/uuid"

	"wordsGo/internal/models"
	"wordsGo/internal/repository"
	"wordsGo/internal/repository/modelsDB"
	"wordsGo/slogger"
)

type DictionaryService interface {
	LoadDictionary(ctx context.Context, path string) error
	GetWords(ctx context.Context, userID uuid.UUID, filter string, limit, page uint64, order string) (*models.ListOfWordsResponse, error)
	SearchWords(ctx context.Context, query string) ([]*models.DictionaryWord, error)
	AddWordToLearning(ctx context.Context, userID uuid.UUID, wordIDStr string) error
	AddWordsByLevel(ctx context.Context, userID uuid.UUID, level string) (int64, error)
	DeleteWordFromLearning(ctx context.Context, userID uuid.UUID, wordIDStr string) error
	UpdateWordDetails(ctx context.Context, userID uuid.UUID, wordIDStr string, req models.UpdateWordRequest) error
	GenerateLesson(ctx context.Context, userID uuid.UUID) (*models.LessonResponse, error)
	ProcessAnswer(ctx context.Context, userID uuid.UUID, req models.SubmitAnswerRequest) (*models.SubmitAnswerResponse, error)
	MarkAsLearned(ctx context.Context, userID uuid.UUID, wordIDStr string) error
	ResetProgress(ctx context.Context, userID uuid.UUID) error
}

type dictionaryService struct {
	repo repository.DictionaryRepository
}

func NewDictionaryService(repo repository.DictionaryRepository) DictionaryService {
	return &dictionaryService{
		repo: repo,
	}
}

func (s *dictionaryService) ResetProgress(ctx context.Context, userID uuid.UUID) error {
	err := s.repo.DeleteAllUserProgress(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to reset progress: %w", err)
	}
	slogger.Log.InfoContext(ctx, "User progress reset", "userID", userID)
	return nil
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

func (s *dictionaryService) GetWords(ctx context.Context, userID uuid.UUID, filter string, limit, page uint64, order string) (*models.ListOfWordsResponse, error) {

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
	wordsDB, total, err := s.repo.GetWords(ctx, userID, filter, order, *pagination)
	if err != nil {
		return nil, err
	}

	wordsResponse := make([]*models.UserWord, len(wordsDB))
	for i, wordModel := range wordsDB {
		baseWord := models.FromDictionaryDB(&wordModel.DictionaryDB)
		wordsResponse[i] = &models.UserWord{
			DictionaryWord:      *baseWord,
			IsLearned:           wordModel.IsLearned,
			CorrectStreak:       wordModel.CorrectStreak,
			TotalMistakes:       wordModel.TotalMistakes,
			DifficultyLevel:     wordModel.DifficultyLevel,
			LastSeen:            wordModel.LastSeen.Format(time.RFC3339),
			CustomTranslation:   wordModel.CustomTranslation,
			CustomTranscription: wordModel.CustomTranscription,
			CustomSynonyms:      wordModel.CustomSynonyms,
		}
	}

	pages := (total + limit - 1) / limit

	resp := &models.ListOfWordsResponse{
		Page:  page,
		Limit: limit,
		Total: total,
		Pages: pages,
		Data:  wordsResponse,
	}

	byLevel, totalProgress, err := s.repo.GetProgressStats(ctx, userID)
	if err == nil {
		resp.Progress = models.ProgressStats{
			Total:   totalProgress,
			ByLevel: byLevel,
		}
	}

	return resp, nil
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
	// 1. Валидация UUID
	wordID, err := uuid.Parse(wordIDStr)
	if err != nil {
		// Если не UUID, пробуем найти слово по 'original' и получить его ID
		slogger.Log.DebugContext(ctx, "wordID is not a UUID, searching by original", "word", wordIDStr)
		words, searchErr := s.repo.SearchByOriginal(ctx, wordIDStr)
		if searchErr != nil || len(words) == 0 {
			return fmt.Errorf("invalid word id format: %w", err)
		}
		// Берем первое совпадение
		wordID = words[0].ID
	}

	// 2. Вызов репозитория
	// Примечание: Можно предварительно проверить существование слова в таблице dictionary,
	// но наличие внешнего ключа (Foreign Key) в БД и так выдаст ошибку, если слова нет.
	err = s.repo.AddWordToUser(ctx, userID, wordID)
	if err != nil {
		return fmt.Errorf("failed to add word to user progress: %w", err)
	}

	return nil
}

func (s *dictionaryService) AddWordsByLevel(ctx context.Context, userID uuid.UUID, level string) (int64, error) {
	count, err := s.repo.AddWordsByLevel(ctx, userID, level)
	if err != nil {
		return 0, fmt.Errorf("failed to bulk add words: %w", err)
	}
	return count, nil
}

func (s *dictionaryService) DeleteWordFromLearning(ctx context.Context, userID uuid.UUID, wordIDStr string) error {
	wordID, err := uuid.Parse(wordIDStr)
	if err != nil {
		return fmt.Errorf("invalid word id format: %w", err)
	}

	err = s.repo.DeleteUserProgress(ctx, userID, wordID)
	if err != nil {
		return fmt.Errorf("failed to delete word from user progress: %w", err)
	}
	return nil
}

func (s *dictionaryService) UpdateWordDetails(ctx context.Context, userID uuid.UUID, wordIDStr string, req models.UpdateWordRequest) error {
	wordID, err := uuid.Parse(wordIDStr)
	if err != nil {
		return fmt.Errorf("invalid word id format: %w", err)
	}

	// Prepare the update model
	// We use ON CONFLICT DO UPDATE, so we just need to pass the new values.
	// If fields are nil, they will remain nil in the DB if we don't handle them carefully.
	// However, usually we want to set them to whatever the user passed (including nil if they cleared it).
	// But our DB helper uses NamedExec. To update only specific fields without overwriting others with zero values
	// is tricky with just one struct.
	// Simplify: We assume the user sends the full state they want for these custom fields.

	p := &modelsDB.UserProgressDB{
		UserID:              userID,
		WordID:              wordID,
		CustomTranslation:   req.Translation,
		CustomTranscription: req.Transcription,
		CustomSynonyms:      req.Synonyms,
		LastSeen:            time.Now(),
	}

	// Note: SaveUserProgress was for learning stats. UpdateUserWordDetails is for content.
	// They share the same table but update different columns (mostly).
	// To be safe and avoid overwriting stats with zeros if the row exists,
	// UpdateUserWordDetails query must ONLY update the custom columns.
	// Let's ensure the repo method does exactly that.
	// Checked repo: It updates custom_* columns and last_seen. Perfect.

	err = s.repo.UpdateUserWordDetails(ctx, p)
	if err != nil {
		return fmt.Errorf("failed to update word details: %w", err)
	}
	return nil
}

// GenerateLesson создает набор слов для "бесконечного" урока.
func (s *dictionaryService) GenerateLesson(ctx context.Context, userID uuid.UUID) (*models.LessonResponse, error) {
	// 1. Пытаемся получить "умный" набор
	wordsDB, err := s.repo.GetLessonWords(ctx, userID)
	if err != nil {
		return nil, err
	}

	targetCount := 10

	// 2. Если слов меньше 10 (например, у новичка нет "сложных" слов), добираем случайными из словаря
	if len(wordsDB) < targetCount {
		excludeIDs := make([]uuid.UUID, 0, len(wordsDB))
		for _, w := range wordsDB {
			excludeIDs = append(excludeIDs, w.ID)
		}

		// Если список пустой (база пустая), нужно избежать ошибки sqlx.In с пустым слайсом
		if len(excludeIDs) == 0 {
			// Добавим несуществующий UUID, чтобы запрос с NOT IN сработал корректно
			excludeIDs = append(excludeIDs, uuid.Nil)
		}

		needed := targetCount - len(wordsDB)
		randomWords, err := s.repo.GetRandomWords(ctx, userID, needed, excludeIDs)
		if err == nil {
			wordsDB = append(wordsDB, randomWords...)
		}
	}

	// 3. Конвертируем в DTO
	respWords := make([]models.WordResponse, len(wordsDB))
	for i, w := range wordsDB {
		transcription := ""
		if w.CustomTranscription != nil && *w.CustomTranscription != "" {
			transcription = *w.CustomTranscription
		} else if w.Transcription != nil {
			transcription = *w.Transcription
		}

		translation := w.Translation
		if w.CustomTranslation != nil && *w.CustomTranslation != "" {
			translation = *w.CustomTranslation
		}

		respWords[i] = models.WordResponse{
			ID:                     w.ID.String(),
			Original:               w.Original,
			Translation:            translation,
			Transcription:          transcription,
			Pos:                    w.Pos,
			Level:                  w.Level,
			PastSimpleSingular:     w.PastSimpleSingular,
			PastSimplePlural:       w.PastSimplePlural,
			PastParticipleSingular: w.PastParticipleSingular,
			PastParticiplePlural:   w.PastParticiplePlural,
			Synonyms:               w.Synonyms,
			DifficultyLevel:        w.DifficultyLevel,
			IsLearned:              w.IsLearned,
		}
	}

	return &models.LessonResponse{Words: respWords}, nil
}

// ProcessAnswer обрабатывает ответ пользователя и обновляет математику сложности.
func (s *dictionaryService) ProcessAnswer(ctx context.Context, userID uuid.UUID, req models.SubmitAnswerRequest) (*models.SubmitAnswerResponse, error) {
	wordID, err := uuid.Parse(req.WordID)
	if err != nil {
		// Если не UUID, пробуем найти слово по 'original' и получить его ID
		slogger.Log.DebugContext(ctx, "wordID is not a UUID in ProcessAnswer, searching by original", "word", req.WordID)
		words, searchErr := s.repo.SearchByOriginal(ctx, req.WordID)
		if searchErr != nil || len(words) == 0 {
			return nil, err
		}
		wordID = words[0].ID
	}

	// 1. Получаем текущий прогресс
	progress, err := s.repo.GetUserProgress(ctx, userID, wordID)
	if err != nil {
		return nil, err
	}

	// Если прогресса нет (первая встреча со словом), инициализируем
	// ... (валидация wordID и получение progress как раньше) ...

	if progress == nil {
		progress = &modelsDB.UserProgressDB{
			UserID: userID, WordID: wordID,
			DifficultyLevel: 0.0, TotalMistakes: 0, CorrectStreak: 0,
		}
	}

	// === НАСТРОЙКИ АЛГОРИТМА ===

	// Вес текущей попытки:
	// Первая попытка в уроке (100%) = 1.0
	// Повторная попытка после ошибки (50%) = 0.5
	weight := 0.5
	if req.IsFirstTry {
		weight = 1.0
	}

	const improvementStep = 0.2 // Шаг снижения сложности
	const penaltyStep = 0.3     // Шаг повышения сложности

	progress.LastSeen = time.Now()

	if req.IsCorrect {
		// --- ПРАВИЛЬНО ---

		// 1. Снижаем сложность
		// Формула: СтараяСложность - (Шаг * Вес)
		change := improvementStep * weight
		progress.DifficultyLevel = math.Max(0.0, progress.DifficultyLevel-change)

		// 2. Стрик (серия правильных ответов)
		// Увеличиваем ТОЛЬКО если это честная первая попытка.
		// Если юзер исправляет ошибку внутри урока — это не считается серией.
		if req.IsFirstTry {
			progress.CorrectStreak++
		}

		// 3. Проверка "Выучено"
		// Условие: Сложность упала до минимума И есть стабильная серия (2 урока подряд)
		if progress.DifficultyLevel <= 0.1 && progress.CorrectStreak >= 2 {
			progress.IsLearned = true
		}

		// Сбрасываем дельту ошибок (для корректного +0 в базе)
		progress.TotalMistakes = 0

	} else {
		// --- ОШИБКА ---

		progress.IsLearned = false

		// 1. Повышаем сложность
		// Формула: СтараяСложность + (Шаг * Вес)
		change := penaltyStep * weight
		progress.DifficultyLevel = math.Min(1.0, progress.DifficultyLevel+change)

		// 2. Сбрасываем серию
		progress.CorrectStreak = 0

		// 3. Фиксируем факт ошибки
		// Это нужно, чтобы GetLessonWords перестал считать слово "новым"
		progress.TotalMistakes = 1
	}

	slogger.Log.DebugContext(ctx, "process answer request", "word", req.WordID, "progress", progress)
	// Сохранение
	err = s.repo.SaveUserProgress(ctx, progress)
	if err != nil {
		return nil, err
	}

	return &models.SubmitAnswerResponse{
		WordID:        wordID.String(),
		NewDifficulty: progress.DifficultyLevel,
		IsLearned:     progress.IsLearned,
		CorrectStreak: progress.CorrectStreak,
	}, nil
}

func (s *dictionaryService) MarkAsLearned(ctx context.Context, userID uuid.UUID, wordIDStr string) error {
	wordID, err := uuid.Parse(wordIDStr)
	if err != nil {
		return err
	}

	progress, err := s.repo.GetUserProgress(ctx, userID, wordID)
	if err != nil {
		return err
	}

	if progress == nil {
		progress = &modelsDB.UserProgressDB{
			UserID: userID,
			WordID: wordID,
		}
	}

	progress.IsLearned = true
	progress.DifficultyLevel = 0.0
	progress.TotalMistakes = 0
	progress.LastSeen = time.Now()

	return s.repo.SaveUserProgress(ctx, progress)
}
