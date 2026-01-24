package modelsDB

import (
	"time"

	"github.com/google/uuid"
)

type DictionaryDB struct {
	ID                     uuid.UUID `db:"id,omitempty"`
	Original               string    `db:"original"`
	Translation            string    `db:"translation"`
	Transcription          string    `db:"transcription"`
	Pos                    string    `db:"pos"`
	Level                  string    `db:"level"`
	PastSimpleSingular     string    `db:"past_simple_singular"`
	PastSimplePlural       string    `db:"past_simple_plural"`
	PastParticipleSingular string    `db:"past_participle_singular"`
	PastParticiplePlural   string    `db:"past_participle_plural"`
	Synonyms               string    `db:"synonyms"`
}

type UserWordDB struct {
	DictionaryDB
	UserID              uuid.UUID `db:"user_id"`
	IsLearned           bool      `db:"is_learned"`
	CorrectStreak       int       `db:"correct_streak"`
	TotalMistakes       int       `db:"total_mistakes"`
	DifficultyLevel     float64   `db:"difficulty_level"`
	LastSeen            time.Time `db:"last_seen"`
	CustomTranslation   *string   `db:"custom_translation"`
	CustomTranscription *string   `db:"custom_transcription"`
	CustomSynonyms      *string   `db:"custom_synonyms"`
}

type WordsDBWithTotal struct {
	UserWordDB
	Total uint64 `json:"total"`
}

// LessonWordDB используется для выборки слов для урока (JOIN dictionary + user_progress).
type LessonWordDB struct {
	ID            uuid.UUID `db:"id"`
	Original      string    `db:"original"`
	Translation   string    `db:"translation"`
	Transcription *string   `db:"transcription"` // Pointer, т.к. может быть NULL

	// Поля из user_progress (могут быть NULL для новых слов)
	UserID          *uuid.UUID `db:"user_id"`
	IsLearned       bool       `db:"is_learned"`
	CorrectStreak   int        `db:"correct_streak"`
	DifficultyLevel float64    `db:"difficulty_level"`
}

// UserProgressDB используется для вставки/обновления прогресса.
type UserProgressDB struct {
	UserID              uuid.UUID `db:"user_id"`
	WordID              uuid.UUID `db:"word_id"`
	IsLearned           bool      `db:"is_learned"`
	CorrectStreak       int       `db:"correct_streak"`
	TotalMistakes       int       `db:"total_mistakes"`
	DifficultyLevel     float64   `db:"difficulty_level"`
	LastSeen            time.Time `db:"last_seen"`
	CustomTranslation   *string   `db:"custom_translation"`
	CustomTranscription *string   `db:"custom_transcription"`
	CustomSynonyms      *string   `db:"custom_synonyms"`
}
