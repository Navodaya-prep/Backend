package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type LiveClass struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Title          string             `bson:"title" json:"title"`
	Subject        string             `bson:"subject" json:"subject"`
	TeacherName    string             `bson:"teacherName" json:"teacherName"`
	Description    string             `bson:"description" json:"description"`
	YouTubeVideoID string             `bson:"youtubeVideoId" json:"youtubeVideoId"`
	ClassLevel     string             `bson:"classLevel" json:"classLevel"`
	Duration       int                `bson:"duration" json:"duration"` // minutes
	IsLive         bool               `bson:"isLive" json:"isLive"`
	IsPremium      bool               `bson:"isPremium" json:"isPremium"`
	StartedAt      time.Time          `bson:"startedAt" json:"startedAt"`
	EndedAt        *time.Time         `bson:"endedAt,omitempty" json:"endedAt,omitempty"`
	CreatedAt      time.Time          `bson:"createdAt" json:"createdAt"`
}

type LiveQuestion struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	LiveClassID  primitive.ObjectID `bson:"liveClassId" json:"liveClassId"`
	Text         string             `bson:"text" json:"text"`
	Options      []string           `bson:"options" json:"options"`
	CorrectIndex int                `bson:"correctIndex" json:"correctIndex"`
	IsReadOnly   bool               `bson:"isReadOnly" json:"isReadOnly"`
	TimerSeconds int                `bson:"timerSeconds" json:"timerSeconds"`
	IsActive     bool               `bson:"isActive" json:"isActive"`
	CreatedAt    time.Time          `bson:"createdAt" json:"createdAt"`
}

type QuizAnswer struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	LiveClassID      primitive.ObjectID `bson:"liveClassId" json:"liveClassId"`
	QuestionID       primitive.ObjectID `bson:"questionId" json:"questionId"`
	UserID           primitive.ObjectID `bson:"userId" json:"userId"`
	UserName         string             `bson:"userName" json:"userName"`
	SelectedIndex    int                `bson:"selectedIndex" json:"selectedIndex"`
	IsCorrect        bool               `bson:"isCorrect" json:"isCorrect"`
	TimeTakenSeconds int                `bson:"timeTakenSeconds" json:"timeTakenSeconds"`
	SubmittedAt      time.Time          `bson:"submittedAt" json:"submittedAt"`
}

type PushToken struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"userId" json:"userId"`
	Token     string             `bson:"token" json:"token"`
	Platform  string             `bson:"platform" json:"platform"` // android | ios
	UpdatedAt time.Time          `bson:"updatedAt" json:"updatedAt"`
}
