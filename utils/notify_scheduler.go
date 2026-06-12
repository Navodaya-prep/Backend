package utils

import (
	"context"
	"log"
	"strconv"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	"github.com/navodayasarthi/api/config"
	"github.com/navodayasarthi/api/models"
)

// istLocation returns the Asia/Kolkata timezone (students are in India).
// Falls back to a fixed +05:30 offset if the tz database is unavailable.
func istLocation() *time.Location {
	if loc, err := time.LoadLocation("Asia/Kolkata"); err == nil {
		return loc
	}
	return time.FixedZone("IST", 5*3600+1800)
}

// StartNotificationScheduler runs the time-based notifications (daily challenge
// reminder, streak-at-risk reminder) at their admin-configured send times.
// Call once from main: go utils.StartNotificationScheduler()
func StartNotificationScheduler() {
	// Seed templates so the admin panel and scheduler always have data.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	SeedNotificationTemplates(ctx)
	cancel()

	log.Println("[notify] scheduler started (timezone: Asia/Kolkata)")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now().In(istLocation())
		hhmm := now.Format("15:04")
		today := now.Format("2006-01-02")

		runScheduledTemplate("daily_challenge", hhmm, today, sendDailyChallengeReminder)
		runScheduledTemplate("streak_risk", hhmm, today, sendStreakRiskReminders)
	}
}

// runScheduledTemplate fires the job when the current time matches the template's
// configured send time. The lastSentDate update is an atomic claim: only one
// process/tick can win it per day, so sends are never duplicated.
func runScheduledTemplate(key, hhmm, today string, job func(ctx context.Context, tpl models.NotificationTemplate, today string)) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	res, err := config.GetCollection("notification_templates").UpdateOne(ctx,
		bson.M{
			"key":          key,
			"enabled":      true,
			"sendTime":     hhmm,
			"lastSentDate": bson.M{"$ne": today},
		},
		bson.M{"$set": bson.M{"lastSentDate": today}},
	)
	if err != nil || res.ModifiedCount == 0 {
		return // not due, disabled, or another instance already claimed today
	}

	tpl, ok := GetNotificationTemplate(ctx, key)
	if !ok {
		return
	}
	job(ctx, tpl, today)
}

// sendDailyChallengeReminder broadcasts the reminder — but only when a challenge
// actually exists for today, so students are never sent to an empty screen.
func sendDailyChallengeReminder(ctx context.Context, tpl models.NotificationTemplate, today string) {
	var challenge models.DailyChallenge
	err := config.GetCollection("daily_challenges").
		FindOne(ctx, bson.M{"date": today}).Decode(&challenge)
	if err != nil {
		log.Printf("[notify] daily_challenge skipped — no challenge for %s", today)
		return
	}

	vars := map[string]string{
		"subject":    challenge.Subject,
		"difficulty": challenge.Difficulty,
	}
	title := RenderNotification(tpl.Title, vars)
	body := RenderNotification(tpl.Body, vars)

	sent := SendPushToAll(ctx, title, body, map[string]string{"screen": "DailyChallenge"})
	LogNotification(ctx, tpl.Key, title, body, "system", sent)
	log.Printf("[notify] daily_challenge sent to %d devices", sent)
}

// sendStreakRiskReminders messages each student who has an active streak but has
// not opened the app today. Personalised per user ({name}, {streak}).
func sendStreakRiskReminders(ctx context.Context, tpl models.NotificationTemplate, today string) {
	startOfDay, _ := time.ParseInLocation("2006-01-02", today, istLocation())

	// Students with a streak worth protecting who were last active before today.
	cursor, err := config.GetCollection("users").Find(ctx, bson.M{
		"streak":         bson.M{"$gte": 2},
		"lastActiveDate": bson.M{"$lt": startOfDay},
	})
	if err != nil {
		log.Printf("[notify] streak_risk user query failed: %v", err)
		return
	}
	var users []models.User
	cursor.All(ctx, &users)
	cursor.Close(ctx)
	if len(users) == 0 {
		return
	}

	// Pull tokens for just these users in one query.
	userIDs := make([]interface{}, 0, len(users))
	for _, u := range users {
		userIDs = append(userIDs, u.ID)
	}
	tokCursor, err := config.GetCollection("pushtokens").
		Find(ctx, bson.M{"userId": bson.M{"$in": userIDs}})
	if err != nil {
		return
	}
	var tokens []models.PushToken
	tokCursor.All(ctx, &tokens)
	tokCursor.Close(ctx)

	tokensByUser := make(map[string][]models.PushToken, len(tokens))
	for _, t := range tokens {
		tokensByUser[t.UserID.Hex()] = append(tokensByUser[t.UserID.Hex()], t)
	}

	var messages []expoPushMessage
	for _, u := range users {
		userTokens := tokensByUser[u.ID.Hex()]
		if len(userTokens) == 0 {
			continue
		}
		vars := map[string]string{"name": u.Name, "streak": strconv.Itoa(u.Streak)}
		messages = append(messages,
			buildMessages(userTokens,
				RenderNotification(tpl.Title, vars),
				RenderNotification(tpl.Body, vars),
				map[string]string{"screen": "Home"})...)
	}
	if len(messages) == 0 {
		return
	}

	sent := sendExpoMessages(ctx, messages)
	LogNotification(ctx, tpl.Key, tpl.Title, tpl.Body, "system", sent)
	log.Printf("[notify] streak_risk sent to %d devices (%d users at risk)", sent, len(users))
}
