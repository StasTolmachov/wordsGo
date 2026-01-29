package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"wordsGo/internal/repository/modelsDB"
)

type DictionaryRepository interface {
	DictionaryInsert(ctx context.Context, dictionary []modelsDB.DictionaryDB) error
	GetWords(ctx context.Context, userID uuid.UUID, filter, order string, pagination modelsDB.Pagination) ([]modelsDB.UserWordDB, uint64, error)
	SearchByOriginal(ctx context.Context, query string) ([]modelsDB.DictionaryDB, error)
	AddWordToUser(ctx context.Context, userID, wordID uuid.UUID) error
	GetLessonWords(ctx context.Context, userID uuid.UUID) ([]modelsDB.LessonWordDB, error)
	GetRandomWords(ctx context.Context, userID uuid.UUID, limit int, excludeIDs []uuid.UUID) ([]modelsDB.LessonWordDB, error)
	GetUserProgress(ctx context.Context, userID, wordID uuid.UUID) (*modelsDB.UserProgressDB, error)
	SaveUserProgress(ctx context.Context, p *modelsDB.UserProgressDB) error
	DeleteUserProgress(ctx context.Context, userID, wordID uuid.UUID) error
	AddWordsByLevel(ctx context.Context, userID uuid.UUID, level string) (int64, error)
	GetProgressStats(ctx context.Context, userID uuid.UUID) (map[string]float64, float64, error)
	UpdateUserWordDetails(ctx context.Context, p *modelsDB.UserProgressDB) error
	DeleteAllUserProgress(ctx context.Context, userID uuid.UUID) error
}

type DictionaryRepo struct {
	db *Postgres
}

func NewDictionaryRepo(db *Postgres) *DictionaryRepo {
	return &DictionaryRepo{db: db}
}

func (r *DictionaryRepo) DeleteAllUserProgress(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM user_progress WHERE user_id = $1`
	_, err := r.db.db.ExecContext(ctx, query, userID)
	return err
}

func (r *DictionaryRepo) DictionaryInsert(ctx context.Context, dictionary []modelsDB.DictionaryDB) error {
	tx, err := r.db.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	query := `insert into dictionary 
    (original, translation, transcription, pos, level, past_simple_singular, past_simple_plural, past_participle_singular, past_participle_plural, Synonyms)
    values (:original, :translation, :transcription, :pos, :level, :past_simple_singular, :past_simple_plural, :past_participle_singular, :past_participle_plural, :synonyms)
    on conflict (original) do nothing`

	batchSize := 5000

	// 3. Проходим по слайсу частями
	for i := 0; i < len(dictionary); i += batchSize {
		end := i + batchSize
		if end > len(dictionary) {
			end = len(dictionary)
		}

		batch := dictionary[i:end]

		// Вставляем текущую пачку
		_, err := tx.NamedExecContext(ctx, query, batch)
		if err != nil {
			return fmt.Errorf("failed to insert batch %d-%d: %w", i, end, err)
		}
	}
	return tx.Commit()
}

func (r *DictionaryRepo) GetWords(ctx context.Context, userID uuid.UUID, filter, order string, pagination modelsDB.Pagination) ([]modelsDB.UserWordDB, uint64, error) {
	sortOrder := "DESC"
	if strings.ToUpper(order) == "ASC" {
		sortOrder = "ASC"
	}

	whereClause := "up.user_id = $1"
	args := []interface{}{userID}
	argIdx := 2

	if filter != "" {
		whereClause += fmt.Sprintf(" AND (d.original ILIKE $%d || '%%' OR d.translation ILIKE $%d || '%%')", argIdx, argIdx)
		args = append(args, filter)
		argIdx++
	}

	query := fmt.Sprintf(`SELECT d.*, 
		up.user_id, up.is_learned, up.correct_streak, up.total_mistakes, up.difficulty_level, up.last_seen,
		up.custom_translation, up.custom_transcription, up.custom_synonyms,
		count(*) OVER() as total
        FROM dictionary d
        JOIN user_progress up ON d.id = up.word_id
        WHERE %s
        ORDER BY d.original %s
        LIMIT $%d OFFSET $%d`, whereClause, sortOrder, argIdx, argIdx+1)

	args = append(args, pagination.Limit, pagination.Offset)

	var wordsDBWithTotal []modelsDB.WordsDBWithTotal
	err := r.db.db.SelectContext(ctx, &wordsDBWithTotal, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all words: %w", err)
	}

	if len(wordsDBWithTotal) == 0 {
		return []modelsDB.UserWordDB{}, 0, nil
	}

	total := wordsDBWithTotal[0].Total

	wordsDB := make([]modelsDB.UserWordDB, len(wordsDBWithTotal))
	for i, words := range wordsDBWithTotal {
		wordsDB[i] = words.UserWordDB
	}

	return wordsDB, total, nil
}

func (r *DictionaryRepo) SearchByOriginal(ctx context.Context, query string) ([]modelsDB.DictionaryDB, error) {
	// Search in both English (original) and Russian (translation)
	sqlQuery := `SELECT * FROM dictionary 
                 WHERE original ILIKE $1 || '%' 
                    OR translation ILIKE $1 || '%'
                 ORDER BY original ASC 
                 LIMIT 20`

	var words []modelsDB.DictionaryDB
	err := r.db.db.SelectContext(ctx, &words, sqlQuery, query)
	if err != nil {
		return nil, fmt.Errorf("failed to search words: %w", err)
	}
	return words, nil
}
func (r *DictionaryRepo) AddWordToUser(ctx context.Context, userID, wordID uuid.UUID) error {
	// Инициализируем прогресс нулями.
	// last_seen ставим в time.Now().
	// В алгоритме выборки (GetLessonWords) используется сортировка ORDER BY last_seen ASC для новых слов.
	// Это реализует очередь (FIFO): слова, добавленные раньше, появятся в уроке первыми.
	query := `INSERT INTO user_progress 
              (user_id, word_id, is_learned, correct_streak, total_mistakes, difficulty_level, last_seen)
              VALUES ($1, $2, false, 0, 0, 0.0, $3)
              ON CONFLICT (user_id, word_id) DO NOTHING`

	_, err := r.db.db.ExecContext(ctx, query, userID, wordID, time.Now())
	return err
}

// GetLessonWords выбирает слова по алгоритму: 2 новых, 5 сложных, 3 на повторение.
// Все выборки происходят ТОЛЬКО из добавленных пользователем слов (user_progress).
func (r *DictionaryRepo) GetLessonWords(ctx context.Context, userID uuid.UUID) ([]modelsDB.LessonWordDB, error) {
	query := `
	WITH 
	new_words AS (
		-- 1. Новые: добавлены в "Мои слова", но еще не выучены и не тренировались (чистый прогресс)
		SELECT d.id, d.original, d.translation, d.transcription, 
		       d.pos, d.level, d.past_simple_singular, d.past_simple_plural, d.past_participle_singular, d.past_participle_plural, d.synonyms,
		       up.user_id, up.is_learned, up.correct_streak, up.difficulty_level,
		       up.custom_translation, up.custom_transcription
		FROM dictionary d
		JOIN user_progress up ON d.id = up.word_id
		WHERE up.user_id = $1 
		  AND up.is_learned = false 
		  AND up.total_mistakes = 0 
		  AND up.correct_streak = 0
		ORDER BY up.last_seen ASC -- Берем те, что были добавлены давным-давно (FIFO)
		LIMIT 2
	),
	hard_words AS (
		-- 2. Сложные/В процессе: не выучены, но уже была попытка (есть ошибки или стрик)
		SELECT d.id, d.original, d.translation, d.transcription, 
		       d.pos, d.level, d.past_simple_singular, d.past_simple_plural, d.past_participle_singular, d.past_participle_plural, d.synonyms,
		       up.user_id, up.is_learned, up.correct_streak, up.difficulty_level,
		       up.custom_translation, up.custom_transcription
		FROM dictionary d
		JOIN user_progress up ON d.id = up.word_id
		WHERE up.user_id = $1 
		  AND up.is_learned = false
		  AND (up.total_mistakes > 0 OR up.correct_streak > 0)
		ORDER BY up.difficulty_level DESC, up.last_seen ASC 
		LIMIT 5
	),
	review_words AS (
		-- 3. Повторение: уже выучены
		SELECT d.id, d.original, d.translation, d.transcription, 
		       d.pos, d.level, d.past_simple_singular, d.past_simple_plural, d.past_participle_singular, d.past_participle_plural, d.synonyms,
		       up.user_id, up.is_learned, up.correct_streak, up.difficulty_level,
		       up.custom_translation, up.custom_transcription
		FROM dictionary d
		JOIN user_progress up ON d.id = up.word_id
		WHERE up.user_id = $1 AND up.is_learned = true
		ORDER BY up.last_seen ASC -- Давно не повторяли
		LIMIT 3
	)
	SELECT * FROM new_words
	UNION ALL
	SELECT * FROM hard_words
	UNION ALL
	SELECT * FROM review_words;
	`

	var words []modelsDB.LessonWordDB
	err := r.db.db.SelectContext(ctx, &words, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to select lesson words: %w", err)
	}
	return words, nil
}

// GetRandomWords добирает случайные слова из списка ПОЛЬЗОВАТЕЛЯ, исключая уже выбранные.
func (r *DictionaryRepo) GetRandomWords(ctx context.Context, userID uuid.UUID, limit int, excludeIDs []uuid.UUID) ([]modelsDB.LessonWordDB, error) {
	if limit <= 0 {
		return nil, nil
	}

	// Важно: JOIN user_progress, чтобы брать только добавленные слова
	query := `
		SELECT d.id, d.original, d.translation, d.transcription, 
		       d.pos, d.level, d.past_simple_singular, d.past_simple_plural, d.past_participle_singular, d.past_participle_plural, d.synonyms,
		       up.user_id, 
		       up.is_learned, 
		       up.correct_streak, 
		       up.difficulty_level,
		       up.custom_translation,
		       up.custom_transcription
		FROM dictionary d
		JOIN user_progress up ON d.id = up.word_id
		WHERE up.user_id = ?
		  AND d.id NOT IN (?)
		ORDER BY RANDOM()
		LIMIT ?
	`

	query, args, err := sqlx.In(query, userID, excludeIDs, limit)
	if err != nil {
		return nil, err
	}

	query = r.db.db.Rebind(query)

	var words []modelsDB.LessonWordDB
	err = r.db.db.SelectContext(ctx, &words, query, args...)
	if err != nil {
		return nil, err
	}
	return words, nil
}

func (r *DictionaryRepo) GetUserProgress(ctx context.Context, userID, wordID uuid.UUID) (*modelsDB.UserProgressDB, error) {
	query := `SELECT * FROM user_progress WHERE user_id = $1 AND word_id = $2`
	var p modelsDB.UserProgressDB
	err := r.db.db.GetContext(ctx, &p, query, userID, wordID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Прогресса нет, это ок
		}
		return nil, err
	}
	return &p, nil
}

// SaveUserProgress сохраняет прогресс. Использует ON CONFLICT для обновления существующих записей.
func (r *DictionaryRepo) SaveUserProgress(ctx context.Context, p *modelsDB.UserProgressDB) error {
	// TotalMistakes суммируется (+ EXCLUDED), чтобы накапливать статистику ошибок.
	// Остальные поля перезаписываются.
	query := `
		INSERT INTO user_progress (
			user_id, word_id, is_learned, correct_streak, total_mistakes, 
			difficulty_level, last_seen
		)
		VALUES (
			:user_id, :word_id, :is_learned, :correct_streak, :total_mistakes, 
			:difficulty_level, :last_seen
		)
		ON CONFLICT (user_id, word_id) DO UPDATE SET
			is_learned = EXCLUDED.is_learned,
			correct_streak = EXCLUDED.correct_streak,
			total_mistakes = user_progress.total_mistakes + EXCLUDED.total_mistakes,
			difficulty_level = EXCLUDED.difficulty_level,
			last_seen = EXCLUDED.last_seen;
	`
	_, err := r.db.db.NamedExecContext(ctx, query, p)
	return err
}

func (r *DictionaryRepo) DeleteUserProgress(ctx context.Context, userID, wordID uuid.UUID) error {
	query := `DELETE FROM user_progress WHERE user_id = $1 AND word_id = $2`
	_, err := r.db.db.ExecContext(ctx, query, userID, wordID)
	return err
}

func (r *DictionaryRepo) AddWordsByLevel(ctx context.Context, userID uuid.UUID, level string) (int64, error) {
	// Insert all words with the given level into user_progress if they don't exist
	query := `
		INSERT INTO user_progress (user_id, word_id, is_learned, correct_streak, total_mistakes, difficulty_level, last_seen)
		SELECT $1, id, false, 0, 0, 0.0, $3
		FROM dictionary
		WHERE level = $2
		ON CONFLICT (user_id, word_id) DO NOTHING
	`
	result, err := r.db.db.ExecContext(ctx, query, userID, level, time.Now())
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}

func (r *DictionaryRepo) GetProgressStats(ctx context.Context, userID uuid.UUID) (map[string]float64, float64, error) {
	query := `
		SELECT 
			level,
			COALESCE(COUNT(*) FILTER (WHERE is_learned = true)::float / NULLIF(COUNT(*), 0)::float * 100, 0) as percent
		FROM dictionary d
		JOIN user_progress up ON d.id = up.word_id
		WHERE up.user_id = $1
		GROUP BY level
	`
	rows, err := r.db.db.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, 0, err
	}
	defer func() {
		_ = rows.Close()
	}()

	byLevel := make(map[string]float64)
	for rows.Next() {
		var level string
		var percent float64
		if err := rows.Scan(&level, &percent); err != nil {
			return nil, 0, err
		}
		if level == "" {
			level = "Other"
		}
		byLevel[level] = percent
	}

	var totalPercent float64
	totalQuery := `
		SELECT COALESCE(COUNT(*) FILTER (WHERE is_learned = true)::float / NULLIF(COUNT(*), 0)::float * 100, 0)
		FROM user_progress
		WHERE user_id = $1
	`
	err = r.db.db.GetContext(ctx, &totalPercent, totalQuery, userID)
	return byLevel, totalPercent, err
}

func (r *DictionaryRepo) UpdateUserWordDetails(ctx context.Context, p *modelsDB.UserProgressDB) error {
	query := `
		INSERT INTO user_progress (user_id, word_id, custom_translation, custom_transcription, custom_synonyms, last_seen, is_learned, correct_streak, total_mistakes, difficulty_level)
		VALUES (:user_id, :word_id, :custom_translation, :custom_transcription, :custom_synonyms, :last_seen, false, 0, 0, 0.0)
		ON CONFLICT (user_id, word_id) DO UPDATE SET
			custom_translation = EXCLUDED.custom_translation,
			custom_transcription = EXCLUDED.custom_transcription,
			custom_synonyms = EXCLUDED.custom_synonyms,
			last_seen = EXCLUDED.last_seen;
	`
	_, err := r.db.db.NamedExecContext(ctx, query, p)
	return err
}
