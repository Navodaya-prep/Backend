package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Doubt struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	UserName  string             `bson:"userName" json:"userName"`
	Subject   string             `bson:"subject" json:"subject"`
	Text      string             `bson:"text" json:"text"`
	ImageURL  string             `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"`
	Status    string             `bson:"status" json:"status"` // "open" | "answered"
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}

type DoubtAnswer struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	DoubtID   primitive.ObjectID `bson:"doubtId" json:"doubtId"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	UserName  string             `bson:"userName" json:"userName"`
	IsAdmin   bool               `bson:"isAdmin" json:"isAdmin"`
	Text      string             `bson:"text" json:"text"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}
