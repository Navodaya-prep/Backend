package utils

import (
	"context"
	"log"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/navodayasarthi/api/config"
	"github.com/navodayasarthi/api/models"
)

// DefaultNotificationTemplates defines every notification type the platform sends.
// These are seeded into MongoDB on first read; admins edit title/body/enabled/sendTime
// from the admin panel. Adding a new type here makes it appear in the panel automatically.
var DefaultNotificationTemplates = []models.NotificationTemplate{
	{
		Key:          "daily_challenge",
		Name:         "Daily Challenge Reminder",
		Description:  "Sent to everyone each morning when a challenge is scheduled for the day. Skipped automatically when no challenge exists.",
		Kind:         "scheduled",
		SendTime:     "09:00",
		Title:        "🧠 Today's challenge is live!",
		Body:         "A new {subject} question is waiting. Can you solve it?",
		Enabled:      true,
		Placeholders: []string{"subject", "difficulty"},
	},
	{
		Key:          "streak_risk",
		Name:         "Streak At Risk",
		Description:  "Sent each evening to students who have a streak going but haven't opened the app today.",
		Kind:         "scheduled",
		SendTime:     "20:00",
		Title:        "🔥 Don't break your streak!",
		Body:         "{name}, your {streak}-day streak ends at midnight. A quick practice session keeps it alive!",
		Enabled:      true,
		Placeholders: []string{"name", "streak"},
	},
	{
		Key:          "streak_milestone",
		Name:         "Streak Milestone",
		Description:  "Sent to a student the moment their streak reaches 7, 30 or 100 days.",
		Kind:         "event",
		Title:        "🏆 {streak}-day streak!",
		Body:         "Amazing consistency, {name}! You've studied {streak} days in a row.",
		Enabled:      true,
		Placeholders: []string{"name", "streak"},
	},
	{
		Key:          "doubt_answered",
		Name:         "Doubt Answered",
		Description:  "Sent to the student who asked a doubt as soon as an admin or teacher answers it.",
		Kind:         "event",
		Title:        "💬 Your doubt has been answered!",
		Body:         "Your {subject} doubt has a reply. Tap to read it.",
		Enabled:      true,
		Placeholders: []string{"subject"},
	},
	{
		Key:          "mock_test_published",
		Name:         "New Mock Test",
		Description:  "Sent to everyone when a new mock test is created.",
		Kind:         "event",
		Title:        "📝 New mock test: {testName}",
		Body:         "A new {subject} test for Class {classLevel} is ready. Attempt it now!",
		Enabled:      true,
		Placeholders: []string{"testName", "subject", "classLevel"},
	},
	{
		Key:          "lesson_added",
		Name:         "New Lesson",
		Description:  "Sent to everyone when a new recorded lesson is published.",
		Kind:         "event",
		Title:        "🎥 New lesson added",
		Body:         "\"{lessonTitle}\" is now available. Start watching!",
		Enabled:      true,
		Placeholders: []string{"lessonTitle"},
	},
	{
		Key:          "live_class_started",
		Name:         "Live Class Started",
		Description:  "Sent to everyone the moment a live class begins.",
		Kind:         "event",
		Title:        "🔴 Live class started!",
		Body:         "{title} — {subject}. Join now!",
		Enabled:      true,
		Placeholders: []string{"title", "subject"},
	},
	{
		Key:          "rank_up",
		Name:         "Leaderboard Rank Up",
		Description:  "Sent to a student when a mock test result improves their weekly leaderboard rank.",
		Kind:         "event",
		Title:        "🎯 You moved up to Rank #{rank}!",
		Body:         "Your score on {testName} pushed you up the leaderboard. Keep it up!",
		Enabled:      true,
		Placeholders: []string{"rank", "testName"},
	},
}

// StreakMilestones are the streak values that trigger the streak_milestone notification.
var StreakMilestones = map[int]bool{7: true, 30: true, 100: true}

// SeedNotificationTemplates inserts any template keys that don't exist yet.
// Existing documents are never overwritten, so admin edits always survive deploys.
func SeedNotificationTemplates(ctx context.Context) {
	col := config.GetCollection("notification_templates")
	for _, tpl := range DefaultNotificationTemplates {
		tpl.ID = primitive.NewObjectID()
		tpl.UpdatedAt = time.Now()
		_, err := col.UpdateOne(ctx,
			bson.M{"key": tpl.Key},
			bson.M{"$setOnInsert": tpl},
			options.Update().SetUpsert(true))
		if err != nil {
			log.Printf("[notify] seed %s failed: %v", tpl.Key, err)
		}
	}
}

// GetNotificationTemplate loads a template by key, falling back to the compiled
// default if the database is unreachable or the key was never seeded.
func GetNotificationTemplate(ctx context.Context, key string) (models.NotificationTemplate, bool) {
	var tpl models.NotificationTemplate
	err := config.GetCollection("notification_templates").
		FindOne(ctx, bson.M{"key": key}).Decode(&tpl)
	if err == nil {
		return tpl, true
	}
	if err != mongo.ErrNoDocuments {
		log.Printf("[notify] load template %s failed: %v", key, err)
	}
	for _, def := range DefaultNotificationTemplates {
		if def.Key == key {
			return def, true
		}
	}
	return tpl, false
}

// RenderNotification substitutes {placeholder} variables into a template string.
func RenderNotification(text string, vars map[string]string) string {
	for k, v := range vars {
		text = strings.ReplaceAll(text, "{"+k+"}", v)
	}
	return text
}

// LogNotification records a send in the history shown in the admin panel.
func LogNotification(ctx context.Context, key, title, body, sentBy string, recipients int) {
	if recipients == 0 {
		return
	}
	if sentBy == "" {
		sentBy = "system"
	}
	entry := models.NotificationLog{
		ID:         primitive.NewObjectID(),
		Key:        key,
		Title:      title,
		Body:       body,
		Recipients: recipients,
		SentBy:     sentBy,
		CreatedAt:  time.Now(),
	}
	if _, err := config.GetCollection("notification_logs").InsertOne(ctx, entry); err != nil {
		log.Printf("[notify] log insert failed: %v", err)
	}
}

// NotifyAll renders the template for key and broadcasts it to every device.
// Runs asynchronously — safe to call from request handlers.
func NotifyAll(key string, vars map[string]string, data map[string]string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		tpl, ok := GetNotificationTemplate(ctx, key)
		if !ok || !tpl.Enabled {
			return
		}
		title := RenderNotification(tpl.Title, vars)
		body := RenderNotification(tpl.Body, vars)

		sent := SendPushToAll(ctx, title, body, data)
		LogNotification(ctx, key, title, body, "system", sent)
		log.Printf("[notify] %s broadcast to %d devices", key, sent)
	}()
}

// NotifyUser renders the template for key and sends it to one user's devices.
// Runs asynchronously — safe to call from request handlers.
func NotifyUser(userID primitive.ObjectID, key string, vars map[string]string, data map[string]string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		tpl, ok := GetNotificationTemplate(ctx, key)
		if !ok || !tpl.Enabled {
			return
		}
		title := RenderNotification(tpl.Title, vars)
		body := RenderNotification(tpl.Body, vars)

		if sent := SendPushToUser(ctx, userID, title, body, data); sent > 0 {
			log.Printf("[notify] %s sent to user %s", key, userID.Hex())
		}
	}()
}

// CheckStreakMilestone sends the milestone notification when a user's streak
// hits one of the configured milestones. Called from UpdateUserActivity.
func CheckStreakMilestone(userID primitive.ObjectID, name string, newStreak int) {
	if !StreakMilestones[newStreak] {
		return
	}
	NotifyUser(userID, "streak_milestone", map[string]string{
		"name":   name,
		"streak": strconv.Itoa(newStreak),
	}, map[string]string{"screen": "Home"})
}
