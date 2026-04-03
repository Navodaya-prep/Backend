package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Lesson is a single piece of content inside a chapter.
// type: "video" | "note"
type Lesson struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ChapterID      primitive.ObjectID `bson:"chapterId" json:"chapterId"`
	CourseID       primitive.ObjectID `bson:"courseId" json:"courseId"`
	Title          string             `bson:"title" json:"title"`
	Type           string             `bson:"type" json:"type"` // "video" | "note"
	YouTubeVideoID string             `bson:"youtubeVideoId,omitempty" json:"youtubeVideoId,omitempty"`
	NoteContent    string             `bson:"noteContent,omitempty" json:"noteContent,omitempty"`
	Description    string             `bson:"description" json:"description"`
	DurationMins   int                `bson:"durationMins" json:"durationMins"`
	Order          int                `bson:"order" json:"order"`
	IsPremium      bool               `bson:"isPremium" json:"isPremium"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
}

// UserCourseProgress tracks which lessons a user has completed in a course.
type UserCourseProgress struct {
	ID                  primitive.ObjectID   `bson:"_id,omitempty" json:"id"`
	UserID              primitive.ObjectID   `bson:"userId" json:"userId"`
	CourseID            primitive.ObjectID   `bson:"courseId" json:"courseId"`
	CompletedLessonIDs  []primitive.ObjectID `bson:"completedLessonIds" json:"completedLessonIds"`
	UpdatedAt           time.Time            `bson:"updatedAt" json:"updatedAt"`
}
