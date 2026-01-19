package models

import "wordsGo/internal/repository/modelsDB"

type DictionaryWord struct {
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

type ListOfWordsResponse struct {
	Page  uint64            `json:"page"`
	Limit uint64            `json:"limit"`
	Total uint64            `json:"total"`
	Pages uint64            `json:"pages"`
	Data  []*DictionaryWord `json:"data"`
}
