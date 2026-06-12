package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/navodayasarthi/api/config"
	"github.com/navodayasarthi/api/models"
)

const (
	expoPushURL   = "https://exp.host/--/api/v2/push/send"
	expoBatchSize = 100 // Expo rejects requests with more than 100 messages
)

type expoPushMessage struct {
	To    string            `json:"to"`
	Title string            `json:"title"`
	Body  string            `json:"body"`
	Data  map[string]string `json:"data,omitempty"`
	Sound string            `json:"sound"`
}

// expoTicket is one entry of Expo's response; order matches the request messages.
type expoTicket struct {
	Status  string `json:"status"` // "ok" | "error"
	Message string `json:"message,omitempty"`
	Details struct {
		Error string `json:"error,omitempty"` // e.g. "DeviceNotRegistered"
	} `json:"details,omitempty"`
}

// sendExpoMessages delivers messages in batches of 100 and prunes tokens that
// Expo reports as DeviceNotRegistered. Returns the number of accepted messages.
func sendExpoMessages(ctx context.Context, messages []expoPushMessage) int {
	sent := 0
	var deadTokens []string

	for start := 0; start < len(messages); start += expoBatchSize {
		end := start + expoBatchSize
		if end > len(messages) {
			end = len(messages)
		}
		batch := messages[start:end]

		payload, err := json.Marshal(batch)
		if err != nil {
			log.Printf("[push] marshal failed: %v", err)
			continue
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, expoPushURL, bytes.NewReader(payload))
		if err != nil {
			log.Printf("[push] request build failed: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("[push] expo push failed: %v", err)
			continue
		}

		var result struct {
			Data []expoTicket `json:"data"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			log.Printf("[push] decode tickets failed: %v", err)
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for i, ticket := range result.Data {
			if ticket.Status == "ok" {
				sent++
				continue
			}
			if ticket.Details.Error == "DeviceNotRegistered" && i < len(batch) {
				deadTokens = append(deadTokens, batch[i].To)
			}
		}
	}

	// Prune tokens for uninstalled / expired devices so we stop sending to them.
	if len(deadTokens) > 0 {
		if _, err := config.GetCollection("pushtokens").
			DeleteMany(ctx, bson.M{"token": bson.M{"$in": deadTokens}}); err == nil {
			log.Printf("[push] pruned %d dead tokens", len(deadTokens))
		}
	}

	return sent
}

func buildMessages(tokens []models.PushToken, title, body string, data map[string]string) []expoPushMessage {
	messages := make([]expoPushMessage, 0, len(tokens))
	for _, t := range tokens {
		messages = append(messages, expoPushMessage{
			To:    t.Token,
			Title: title,
			Body:  body,
			Data:  data,
			Sound: "default",
		})
	}
	return messages
}

// SendPushToAll sends one notification to every registered device.
// Returns the number of devices the message was accepted for.
func SendPushToAll(ctx context.Context, title, body string, data map[string]string) int {
	cursor, err := config.GetCollection("pushtokens").Find(ctx, bson.M{})
	if err != nil {
		log.Printf("[push] failed to fetch tokens: %v", err)
		return 0
	}
	defer cursor.Close(ctx)

	var tokens []models.PushToken
	cursor.All(ctx, &tokens)
	if len(tokens) == 0 {
		return 0
	}

	return sendExpoMessages(ctx, buildMessages(tokens, title, body, data))
}

// SendPushToUser sends one notification to all devices of a single user.
func SendPushToUser(ctx context.Context, userID primitive.ObjectID, title, body string, data map[string]string) int {
	cursor, err := config.GetCollection("pushtokens").Find(ctx, bson.M{"userId": userID})
	if err != nil {
		return 0
	}
	defer cursor.Close(ctx)

	var tokens []models.PushToken
	cursor.All(ctx, &tokens)
	if len(tokens) == 0 {
		return 0
	}

	return sendExpoMessages(ctx, buildMessages(tokens, title, body, data))
}

// SendLiveClassNotification pushes a notification to all registered users when a class starts.
// Message content comes from the admin-editable "live_class_started" template.
func SendLiveClassNotification(classID, title, subject string) {
	NotifyAll("live_class_started",
		map[string]string{"title": title, "subject": subject},
		map[string]string{"screen": "LiveClasses", "classId": classID},
	)
}

// UpsertPushToken saves or updates a user's Expo push token.
func UpsertPushToken(userID primitive.ObjectID, token, platform string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"userId": userID}
	update := bson.M{
		"$set": bson.M{
			"userId":    userID,
			"token":     token,
			"platform":  platform,
			"updatedAt": time.Now(),
		},
		"$setOnInsert": bson.M{"_id": primitive.NewObjectID()},
	}

	opts := options.Update().SetUpsert(true)
	_, err := config.GetCollection("pushtokens").UpdateOne(ctx, filter, update, opts)
	return err
}
