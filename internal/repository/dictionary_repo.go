package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"wordsGo/internal/repository/modelsDB"
)

type DictionaryRepo struct {
	db *Postgres
}

func NewDictionaryRepo(db *Postgres) *DictionaryRepo {
	return &DictionaryRepo{db: db}
}

func (r *DictionaryRepo) DictionaryInsert(ctx context.Context, dictionary []modelsDB.DictionaryDB) error {
	tx, err := r.db.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

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

func (r *DictionaryRepo) GetWords(ctx context.Context, userID uuid.UUID, order string, pagination modelsDB.Pagination) ([]modelsDB.DictionaryDB, uint64, error) {
	sortOrder := "DESC"
	if strings.ToUpper(order) == "ASC" {
		sortOrder = "ASC"
	}

	query := fmt.Sprintf(`SELECT d.*, count(*) OVER() as total
        FROM dictionary d
        JOIN user_progress up ON d.id = up.word_id
        WHERE up.user_id = $1
        ORDER BY d.original %s
        LIMIT $2 OFFSET $3`, sortOrder)

	var wordsDBWithTotal []modelsDB.WordsDBWithTotal
	err := r.db.db.SelectContext(ctx, &wordsDBWithTotal, query, userID, pagination.Limit, pagination.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all words: %w", err)
	}

	if len(wordsDBWithTotal) == 0 {
		return []modelsDB.DictionaryDB{}, 0, nil
	}

	total := wordsDBWithTotal[0].Total

	wordsDB := make([]modelsDB.DictionaryDB, len(wordsDBWithTotal))
	for i, words := range wordsDBWithTotal {
		wordsDB[i] = words.DictionaryDB
	}

	return wordsDB, total, nil
}

func (r *DictionaryRepo) SearchByOriginal(ctx context.Context, query string) ([]modelsDB.DictionaryDB, error) {
	// Ищем слова, начинающиеся на query (например, "app%"). Лимит 10-20, чтобы не грузить лишнее.
	sqlQuery := `SELECT * FROM dictionary 
                 WHERE original ILIKE $1 || '%' 
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
	// Инициализируем прогресс нулями. last_seen ставим в прошлое или null, чтобы слово сразу могло попасть в урок.
	query := `INSERT INTO user_progress 
              (user_id, word_id, is_learned, correct_streak, total_mistakes, DifficultyLevel, last_seen)
              VALUES ($1, $2, false, 0, 0, 0.0, NOW())`

	// Если вы добавили уникальный индекс, можно использовать ON CONFLICT DO NOTHING,
	// чтобы не возвращать ошибку, если слово уже добавлено.

	_, err := r.db.db.ExecContext(ctx, query, userID, wordID)
	return err
}
