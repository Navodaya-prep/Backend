package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// DailyChallenge represents a single question posted for a specific date
type DailyChallenge struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Date         string             `bson:"date" json:"date"` // "YYYY-MM-DD"
	Text         string             `bson:"text" json:"text"`
	Options      []string           `bson:"options" json:"options"`
	CorrectIndex int                `bson:"correctIndex" json:"correctIndex"`
	Explanation  string             `bson:"explanation" json:"explanation"`
	Subject      string             `bson:"subject" json:"subject"`
	Difficulty   string             `bson:"difficulty" json:"difficulty"`
	CreatedBy    string             `bson:"createdBy" json:"createdBy"` // admin email
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
}

// DailyChallengeAttempt tracks a user's attempt at the daily challenge
type DailyChallengeAttempt struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID        primitive.ObjectID `bson:"userId" json:"userId"`
	ChallengeID   primitive.ObjectID `bson:"challengeId" json:"challengeId"`
	Date          string             `bson:"date" json:"date"` // "YYYY-MM-DD"
	SelectedIndex int                `bson:"selectedIndex" json:"selectedIndex"`
	IsCorrect     bool               `bson:"isCorrect" json:"isCorrect"`
	Points        int                `bson:"points" json:"points"`
	Attempts      int                `bson:"attempts" json:"attempts"`       // number of attempts made
	Revealed      bool               `bson:"revealed" json:"revealed"`       // if user chose to reveal answer
	TimeTakenMs   int64              `bson:"timeTakenMs" json:"timeTakenMs"` // time to first correct answer in milliseconds
	SolvedAt      *time.Time         `bson:"solvedAt,omitempty" json:"solvedAt,omitempty"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt     time.Time          `bson:"updatedAt" json:"updatedAt"`
}
