package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Question struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ChapterID    *primitive.ObjectID `bson:"chapterId,omitempty" json:"chapterId,omitempty"`
	Subject      string              `bson:"subject" json:"subject"`
	Text         string              `bson:"text" json:"text"`
	Options      []string            `bson:"options" json:"options"`
	CorrectIndex int                 `bson:"correctIndex" json:"correctIndex"`
	Explanation  string              `bson:"explanation" json:"explanation"`
	Difficulty   string              `bson:"difficulty" json:"difficulty"`
	ClassLevel   string              `bson:"classLevel" json:"classLevel"`
	IsPremium    bool                `bson:"isPremium" json:"isPremium"`
	Tags         []string            `bson:"tags" json:"tags"`
	CreatedAt    time.Time           `bson:"createdAt" json:"createdAt"`
}
