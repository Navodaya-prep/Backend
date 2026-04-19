package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Bookmark struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     primitive.ObjectID `bson:"userId" json:"userId"`
	QuestionID primitive.ObjectID `bson:"questionId" json:"questionId"`
	Source     string             `bson:"source" json:"source"` // "practice" | "mocktest"
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
}
