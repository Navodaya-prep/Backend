package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ContactMessage struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name      string             `bson:"name" json:"name"`
	Phone     string             `bson:"phone" json:"phone"`
	Message   string             `bson:"message" json:"message"`
	CreatedAt time.Time          `bson:"createdAt" json:"createdAt"`
}
