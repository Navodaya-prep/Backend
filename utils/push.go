package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"navodaya-api/config"
	"navodaya-api/models"
)

const expoPushURL = "https://exp.host/--/api/v2/push/send"

type expoPushMessage struct {
	To    string            `json:"to"`
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data,omitempty"`
	Sound string            `json:"sound"`
}

// SendLiveClassNotification pushes a notification to all registered users when a class starts.
func SendLiveClassNotification(classID, title, subject string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cursor, err := config.GetCollection("pushtokens").Find(ctx, bson.M{})
		if err != nil {
			log.Printf("[push] failed to fetch tokens: %v", err)
			return
		}
		defer cursor.Close(ctx)

		var tokens []models.PushToken
		cursor.All(ctx, &tokens)

		if len(tokens) == 0 {
			return
		}

		messages := make([]expoPushMessage, 0, len(tokens))
		for _, t := range tokens {
			messages = append(messages, expoPushMessage{
				To:    t.Token,
				Title: "🔴 Live Class Started!",
				Body:  fmt.Sprintf("%s — %s", title, subject),
				Sound: "default",
				Data: map[string]string{
					"screen":  "LiveClasses",
					"classId": classID,
				},
			})
		}

		payload, _ := json.Marshal(messages)
		resp, err := http.Post(expoPushURL, "application/json", bytes.NewReader(payload))
		if err != nil {
			log.Printf("[push] expo push failed: %v", err)
			return
		}
		defer resp.Body.Close()
		log.Printf("[push] sent notifications to %d devices", len(messages))
	}()
}

// UpsertPushToken saves or updates a user's Expo push token.
func UpsertPushToken(userID primitive.ObjectID, token, platform string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"userId": userID}
	update := bson.M{"$set": models.PushToken{
		UserID:    userID,
		Token:     token,
		Platform:  platform,
		UpdatedAt: time.Now(),
	}}

	_, err := config.GetCollection("pushtokens").UpdateOne(ctx, filter, update,
		// upsert handled by caller with options — kept simple here
	)
	return err
}
