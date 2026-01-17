package models

import (
	"time"

	"github.com/google/uuid"
)

type Dictionary struct {
	ID            uuid.UUID
	Original      string
	Translation   string
	Transcription string
}

type Users struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	FirstName    string
	LastName     string
}

type UsersProgress struct {
	UserID        uuid.UUID
	WordID        uuid.UUID
	IsLearned     bool
	correctStreak int
	TotalMistakes int
	LastSeen      time.Time
}
