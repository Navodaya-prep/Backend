package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserChapterProgress tracks which questions a user has solved in a chapter.
type UserChapterProgress struct {
	ID                primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID            primitive.ObjectID   `bson:"userId" json:"userId"`
	ChapterID         primitive.ObjectID   `bson:"chapterId" json:"chapterId"`
	SolvedQuestionIDs []primitive.ObjectID `bson:"solvedQuestionIds" json:"solvedQuestionIds"`
	UpdatedAt         time.Time            `bson:"updatedAt" json:"updatedAt"`
}
