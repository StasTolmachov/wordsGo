package models

import "wordsGo/internal/repository/modelsDB"

type DictionaryWord struct {
	ID                     string `json:"id"`
	Original               string `json:"original"`
	Translation            string `json:"translation"`
	Transcription          string `json:"transcription"`
	Pos                    string `json:"pos"`
	Level                  string `json:"level"`
	PastSimpleSingular     string `json:"past_simple_singular,omitempty"`
	PastSimplePlural       string `json:"past_simple_plural,omitempty"`
	PastParticipleSingular string `json:"past_participle_singular,omitempty"`
	PastParticiplePlural   string `json:"past_participle_plural,omitempty"`
	Synonyms               string `json:"synonyms,omitempty"`
}

// Конвертер из JSON-модели в DB-модель
func ToDictionaryDB(word DictionaryWord) modelsDB.DictionaryDB {
	return modelsDB.DictionaryDB{
		Original:               word.Original,
		Translation:            word.Translation,
		Transcription:          word.Transcription,
		Pos:                    word.Pos,
		Level:                  word.Level,
		PastSimpleSingular:     word.PastSimpleSingular,
		PastSimplePlural:       word.PastSimplePlural,
		PastParticipleSingular: word.PastParticipleSingular,
		PastParticiplePlural:   word.PastParticiplePlural,
		Synonyms:               word.Synonyms,
	}
}

func FromDictionaryDB(word *modelsDB.DictionaryDB) *DictionaryWord {
	return &DictionaryWord{
		ID:                     word.ID.String(),
		Original:               word.Original,
		Translation:            word.Translation,
		Transcription:          word.Transcription,
		Pos:                    word.Pos,
		Level:                  word.Level,
		PastSimpleSingular:     word.PastSimpleSingular,
		PastSimplePlural:       word.PastSimplePlural,
		PastParticipleSingular: word.PastParticipleSingular,
		PastParticiplePlural:   word.PastParticiplePlural,
		Synonyms:               word.Synonyms,
	}
}

type UserWord struct {
	DictionaryWord
	IsLearned           bool    `json:"is_learned"`
	CorrectStreak       int     `json:"correct_streak"`
	TotalMistakes       int     `json:"total_mistakes"`
	DifficultyLevel     float64 `json:"difficulty_level"`
	LastSeen            string  `json:"last_seen"`
	CustomTranslation   *string `json:"custom_translation,omitempty"`
	CustomTranscription *string `json:"custom_transcription,omitempty"`
	CustomSynonyms      *string `json:"custom_synonyms,omitempty"`
}

type UpdateWordRequest struct {
	Translation   *string `json:"translation"`
	Transcription *string `json:"transcription"`
	Synonyms      *string `json:"synonyms"`
}

type ProgressStats struct {
	Total   float64            `json:"total"`
	ByLevel map[string]float64 `json:"by_level"`
}

type ListOfWordsResponse struct {
	Page     uint64        `json:"page"`
	Limit    uint64        `json:"limit"`
	Total    uint64        `json:"total"`
	Pages    uint64        `json:"pages"`
	Data     []*UserWord   `json:"data"`
	Progress ProgressStats `json:"progress"`
}

// WordResponse описывает слово, которое отправляется клиенту для урока.
type WordResponse struct {
	ID                     string  `json:"id"`
	Original               string  `json:"original"`
	Translation            string  `json:"translation"`
	Transcription          string  `json:"transcription,omitempty"`
	Pos                    string  `json:"pos,omitempty"`
	Level                  string  `json:"level,omitempty"`
	PastSimpleSingular     string  `json:"past_simple_singular,omitempty"`
	PastSimplePlural       string  `json:"past_simple_plural,omitempty"`
	PastParticipleSingular string  `json:"past_participle_singular,omitempty"`
	PastParticiplePlural   string  `json:"past_participle_plural,omitempty"`
	Synonyms               string  `json:"synonyms,omitempty"`
	DifficultyLevel        float64 `json:"difficulty_level"`
	IsLearned              bool    `json:"is_learned"`
}

// LessonResponse содержит список слов для локального цикла на клиенте.
type LessonResponse struct {
	Words []WordResponse `json:"words"`
}

// SubmitAnswerRequest — запрос от клиента при ответе на слово.
type SubmitAnswerRequest struct {
	WordID    string `json:"word_id" validate:"required"`
	IsCorrect bool   `json:"is_correct"`
}

// SubmitAnswerResponse — ответ сервера с обновленными метриками слова.
type SubmitAnswerResponse struct {
	WordID        string  `json:"word_id"`
	NewDifficulty float64 `json:"new_difficulty"`
	IsLearned     bool    `json:"is_learned"`
	CorrectStreak int     `json:"correct_streak"`
}
