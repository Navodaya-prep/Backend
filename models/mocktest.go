package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MockTest struct {
	ID           primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	Title        string               `bson:"title" json:"title"`
	Subject      string               `bson:"subject" json:"subject"`
	Duration     int                  `bson:"duration" json:"duration"` // minutes
	TotalMarks   int                  `bson:"totalMarks" json:"totalMarks"`
	ClassLevel   string               `bson:"classLevel" json:"classLevel"`
	QuestionIDs  []primitive.ObjectID `bson:"questions" json:"-"`
	Questions    []Question           `bson:"-" json:"questions,omitempty"`
	IsPremium    bool                 `bson:"isPremium" json:"isPremium"`
	AttemptCount int                  `bson:"attemptCount" json:"attemptCount"`
	CreatedAt    time.Time            `bson:"createdAt" json:"createdAt"`
}

type AttemptAnswer struct {
	QuestionID    primitive.ObjectID `bson:"questionId" json:"questionId"`
	SelectedIndex int                `bson:"selectedIndex" json:"selectedIndex"`
	IsCorrect     bool               `bson:"isCorrect" json:"isCorrect"`
}

type MockTestAttempt struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"userId" json:"userId"`
	MockTestID  primitive.ObjectID `bson:"mockTestId" json:"mockTestId"`
	Answers     []AttemptAnswer    `bson:"answers" json:"answers"`
	Score       int                `bson:"score" json:"score"`
	TotalMarks  int                `bson:"totalMarks" json:"totalMarks"`
	TimeTaken   int                `bson:"timeTaken" json:"timeTaken"` // seconds
	CompletedAt time.Time          `bson:"completedAt" json:"completedAt"`
}
