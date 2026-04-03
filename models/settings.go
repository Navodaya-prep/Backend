package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Settings struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ExamDate  time.Time          `bson:"examDate" json:"examDate"`
	ExamName  string             `bson:"examName" json:"examName"`
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy primitive.ObjectID `bson:"updatedBy" json:"updatedBy"`
}
