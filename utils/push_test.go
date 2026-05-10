package utils

import (
	"context"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"github.com/navodayaprime/api/config"
)

// ─── UpsertPushToken ──────────────────────────────────────────────────────────

func TestUpsertPushToken_Success(t *testing.T) {
	requireDB(t)
	clearCollection(t, "pushtokens")

	userID := primitive.NewObjectID()
	err := UpsertPushToken(userID, "expo-push-token-abc", "android")
	if err != nil {
		t.Errorf("UpsertPushToken: unexpected error: %v", err)
	}
}

func TestUpsertPushToken_UpdatesExisting(t *testing.T) {
	requireDB(t)
	clearCollection(t, "pushtokens")

	userID := primitive.NewObjectID()

	if err := UpsertPushToken(userID, "old-token", "android"); err != nil {
		t.Fatalf("first UpsertPushToken: %v", err)
	}

	if err := UpsertPushToken(userID, "new-token", "ios"); err != nil {
		t.Errorf("second UpsertPushToken: %v", err)
	}

	count, err := config.GetCollection("pushtokens").CountDocuments(context.Background(), map[string]interface{}{"userId": userID})
	if err != nil {
		t.Fatalf("count error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 push token doc after upsert, got %d", count)
	}
}

func TestUpsertPushToken_DifferentUsers(t *testing.T) {
	requireDB(t)
	clearCollection(t, "pushtokens")

	u1 := primitive.NewObjectID()
	u2 := primitive.NewObjectID()

	UpsertPushToken(u1, "token-1", "android")
	UpsertPushToken(u2, "token-2", "ios")

	count, _ := config.GetCollection("pushtokens").CountDocuments(context.Background(), map[string]interface{}{})
	if count != 2 {
		t.Errorf("expected 2 tokens for 2 users, got %d", count)
	}
}
