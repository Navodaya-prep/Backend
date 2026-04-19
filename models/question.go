package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// QuestionOption represents a single answer option which can be text or image.
// Type: "text" (default) or "image"
// Value: the text content or the image URL
type QuestionOption struct {
	Type  string `bson:"type" json:"type"`   // "text" or "image"
	Value string `bson:"value" json:"value"` // text content or image URL
}

type Question struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ChapterID    *primitive.ObjectID `bson:"chapterId,omitempty" json:"chapterId,omitempty"`
	Subject      string              `bson:"subject" json:"subject"`
	Text         string              `bson:"text" json:"text"`
	ImageURL     string              `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"` // optional image for question
	Options      []QuestionOption    `bson:"options" json:"options"`
	CorrectIndex int                 `bson:"correctIndex" json:"correctIndex"`
	Explanation  string              `bson:"explanation" json:"explanation"`
	Difficulty   string              `bson:"difficulty" json:"difficulty"`
	ClassLevel   string              `bson:"classLevel" json:"classLevel"`
	IsPremium    bool                `bson:"isPremium" json:"isPremium"`
	IsPYQ        bool                `bson:"isPYQ" json:"isPYQ"`       // Previous Year Question
	ExamYear     string              `bson:"examYear" json:"examYear"` // e.g., "2024", "2023"
	Tags         []string            `bson:"tags" json:"tags"`
	CreatedAt    time.Time           `bson:"createdAt" json:"createdAt"`
}
