package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Course struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title         string             `bson:"title" json:"title"`
	Subject       string             `bson:"subject" json:"subject"`
	ClassLevel    string             `bson:"classLevel" json:"classLevel"`
	Thumbnail     string             `bson:"thumbnail" json:"thumbnail"`
	Description   string             `bson:"description" json:"description"`
	ChaptersCount int                `bson:"chaptersCount" json:"chaptersCount"`
	VideosCount   int                `bson:"videosCount" json:"videosCount"`
	IsPremium     bool               `bson:"isPremium" json:"isPremium"`
	Order         int                `bson:"order" json:"order"`
	CreatedAt     time.Time          `bson:"createdAt" json:"createdAt"`
}

type Chapter struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	CourseID    primitive.ObjectID `bson:"courseId" json:"courseId"`
	Title       string             `bson:"title" json:"title"`
	Description string             `bson:"description" json:"description"`
	Order       int                `bson:"order" json:"order"`
	IsPremium   bool               `bson:"isPremium" json:"isPremium"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
}
