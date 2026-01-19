package repository

import (
	"context"
	"fmt"

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
    (original, translation, transcription, pos, level, PastSimpleSingular, PastSimplePlural, PastParticipleSingular, PastParticiplePlural, Synonyms)
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
