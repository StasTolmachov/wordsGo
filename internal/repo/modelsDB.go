package repo

import (
	"time"

	"github.com/google/uuid"
)

type Dictionary struct {
	ID            uuid.UUID `db:"id,omitempty"`
	Original      string    `db:"original"`
	Translation   string    `db:"translation"`
	Transcription string    `db:"transcription"`
}

type Users struct {
	ID           uuid.UUID `db:"id,omitempty"`
	Email        string    `db:"email"`
	PasswordHash string    `db:"password_hash"`
	FirstName    string    `db:"first_name"`
	LastName     string    `db:"last_name"`
}

type UsersProgress struct {
	UserID        uuid.UUID `db:"user_id,omitempty"`
	WordID        uuid.UUID `db:"word_id,omitempty"`
	IsLearned     bool      `db:"is_learned"`
	correctStreak int       `db:"correct_streak"`
	TotalMistakes int       `db:"total_mistakes"`
	LastSeen      time.Time `db:"last_seen"`
}
