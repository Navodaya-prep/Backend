package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name           string             `bson:"name" json:"name"`
	Phone          string             `bson:"phone" json:"phone"`
	ClassLevel     string             `bson:"classLevel" json:"classLevel"`
	State          string             `bson:"state" json:"state"`
	StarPoints     int                `bson:"starPoints" json:"starPoints"`
	Streak         int                `bson:"streak" json:"streak"`
	IsPremium      bool               `bson:"isPremium" json:"isPremium"`
	IsAdmin        bool               `bson:"isAdmin" json:"isAdmin"`
	LastActiveDate *time.Time         `bson:"lastActiveDate,omitempty" json:"lastActiveDate,omitempty"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt      time.Time          `bson:"updatedAt" json:"updatedAt"`
}
