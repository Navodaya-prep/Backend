package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Teacher struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FirstName  string             `bson:"firstName" json:"firstName"`
	LastName   string             `bson:"lastName" json:"lastName"`
	Email      string             `bson:"email" json:"email"`
	Password   string             `bson:"password" json:"-"`
	Phone      string             `bson:"phone" json:"phone"`
	Subject    string             `bson:"subject" json:"subject"`
	ClassLevel string             `bson:"classLevel" json:"classLevel"`
	Bio        string             `bson:"bio" json:"bio"`
	IsActive   bool               `bson:"isActive" json:"isActive"`
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt  time.Time          `bson:"updatedAt" json:"updatedAt"`
}
