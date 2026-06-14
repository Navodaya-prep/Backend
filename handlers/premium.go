package handlers

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/navodayasarthi/api/config"
	"github.com/navodayasarthi/api/models"
)

// userIsPremium reports whether the given user currently has premium access.
// On any lookup error it returns false (fail-closed: treat as non-premium).
func userIsPremium(userID primitive.ObjectID) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := config.GetCollection("users").FindOne(ctx,
		bson.M{"_id": userID}).Decode(&user); err != nil {
		return false
	}
	return user.IsPremium
}

// courseIsPremium reports whether a course is marked premium. Used to lock a
// whole course (and therefore all its chapters/lessons) for non-premium users.
func courseIsPremium(courseID primitive.ObjectID) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var course models.Course
	if err := config.GetCollection("courses").FindOne(ctx,
		bson.M{"_id": courseID}).Decode(&course); err != nil {
		return false
	}
	return course.IsPremium
}

// redactPremiumLessons strips the video/note content from premium lessons when
// the requesting user is not premium, so locked content can't be played even
// if the client ignores the lock. Title/flags are preserved for the teaser.
func redactPremiumLessons(lessons []models.Lesson, isPremium bool) {
	if isPremium {
		return
	}
	for i := range lessons {
		if lessons[i].IsPremium {
			lessons[i].YouTubeVideoID = ""
			lessons[i].NoteContent = ""
		}
	}
}

// redactPremiumQuestions strips answer-revealing fields (options, correct
// index, explanation) from premium questions when the requesting user is not
// premium. The question's text/difficulty/flags are preserved so the client
// can still render a visible-but-locked teaser. Free questions are untouched.
func redactPremiumQuestions(questions []models.Question, isPremium bool) {
	if isPremium {
		return
	}
	for i := range questions {
		if questions[i].IsPremium {
			questions[i].Options = []models.QuestionOption{}
			questions[i].CorrectIndex = -1
			questions[i].Explanation = ""
		}
	}
}
