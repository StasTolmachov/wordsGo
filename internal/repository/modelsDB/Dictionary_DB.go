package modelsDB

import "github.com/google/uuid"

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

type WordsDBWithTotal struct {
	DictionaryDB
	Total uint64 `json:"total"`
}
