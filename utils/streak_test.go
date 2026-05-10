package utils

import (
	"context"
	"testing"
	"time"

	"navodaya-api/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ─── CalculateStreak (pure function) ─────────────────────────────────────────

func TestCalculateStreak_NilLastActive(t *testing.T) {
	// First activity ever — should return 1
	result := CalculateStreak(nil, 0)
	if result != 1 {
		t.Errorf("want 1, got %d", result)
	}
}

func TestCalculateStreak_NilLastActive_NonZeroStreak(t *testing.T) {
	// Even with stored streak, nil date resets to 1
	result := CalculateStreak(nil, 5)
	if result != 1 {
		t.Errorf("want 1, got %d", result)
	}
}

func TestCalculateStreak_SameDay(t *testing.T) {
	now := time.Now()
	// Same day: streak should be unchanged
	result := CalculateStreak(&now, 7)
	if result != 7 {
		t.Errorf("want 7 (same day), got %d", result)
	}
}

func TestCalculateStreak_SameDay_StartOfDay(t *testing.T) {
	today := time.Now()
	startOfDay := time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location())
	result := CalculateStreak(&startOfDay, 3)
	if result != 3 {
		t.Errorf("want 3 (start of same day), got %d", result)
	}
}

func TestCalculateStreak_SameDay_EndOfDay(t *testing.T) {
	today := time.Now()
	endOfDay := time.Date(today.Year(), today.Month(), today.Day(), 23, 59, 59, 0, today.Location())
	result := CalculateStreak(&endOfDay, 10)
	if result != 10 {
		t.Errorf("want 10 (end of same day), got %d", result)
	}
}

func TestCalculateStreak_ConsecutiveDay(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	result := CalculateStreak(&yesterday, 5)
	if result != 6 {
		t.Errorf("want 6 (consecutive day), got %d", result)
	}
}

func TestCalculateStreak_ConsecutiveDay_FromOne(t *testing.T) {
	yesterday := time.Now().AddDate(0, 0, -1)
	result := CalculateStreak(&yesterday, 1)
	if result != 2 {
		t.Errorf("want 2 (streak 1 → 2), got %d", result)
	}
}

func TestCalculateStreak_TwoDaysAgo(t *testing.T) {
	twoDaysAgo := time.Now().AddDate(0, 0, -2)
	result := CalculateStreak(&twoDaysAgo, 10)
	if result != 1 {
		t.Errorf("want 1 (streak broken at 2 days gap), got %d", result)
	}
}

func TestCalculateStreak_OneWeekAgo(t *testing.T) {
	oneWeekAgo := time.Now().AddDate(0, 0, -7)
	result := CalculateStreak(&oneWeekAgo, 20)
	if result != 1 {
		t.Errorf("want 1 (long gap resets streak), got %d", result)
	}
}

func TestCalculateStreak_OneYearAgo(t *testing.T) {
	oneYearAgo := time.Now().AddDate(-1, 0, 0)
	result := CalculateStreak(&oneYearAgo, 365)
	if result != 1 {
		t.Errorf("want 1 (year gap resets streak), got %d", result)
	}
}

// ─── UpdateUserActivity (requires MongoDB) ────────────────────────────────────

func TestUpdateUserActivity_Success(t *testing.T) {
	requireDB(t)
	clearCollection(t, "users")

	userID := primitive.NewObjectID()
	yesterday := time.Now().AddDate(0, 0, -1)
	user := models.User{
		ID:             userID,
		Name:           "Test User",
		Phone:          "9876543210",
		Streak:         5,
		LastActiveDate: &yesterday,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("users")
	col.InsertOne(ctx, user)

	err := UpdateUserActivity(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify streak was incremented
	var updated models.User
	col.FindOne(ctx, map[string]interface{}{"_id": userID}).Decode(&updated)
	if updated.Streak != 6 {
		t.Errorf("want streak=6, got %d", updated.Streak)
	}
}

func TestUpdateUserActivity_NotFound(t *testing.T) {
	requireDB(t)
	clearCollection(t, "users")

	nonExistentID := primitive.NewObjectID()
	err := UpdateUserActivity(context.Background(), nonExistentID)
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}

func TestUpdateUserActivity_ResetStreak(t *testing.T) {
	requireDB(t)
	clearCollection(t, "users")

	userID := primitive.NewObjectID()
	oldDate := time.Now().AddDate(0, 0, -5)
	user := models.User{
		ID:             userID,
		Name:           "Test",
		Phone:          "9999999999",
		Streak:         15,
		LastActiveDate: &oldDate,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("users")
	col.InsertOne(ctx, user)

	err := UpdateUserActivity(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var updated models.User
	col.FindOne(ctx, map[string]interface{}{"_id": userID}).Decode(&updated)
	if updated.Streak != 1 {
		t.Errorf("want streak=1 (reset), got %d", updated.Streak)
	}
}

// ─── GetUserStreak (requires MongoDB) ────────────────────────────────────────

func TestGetUserStreak_SameDay(t *testing.T) {
	requireDB(t)
	clearCollection(t, "users")

	userID := primitive.NewObjectID()
	now := time.Now()
	user := models.User{
		ID:             userID,
		Name:           "Test",
		Phone:          "8888888888",
		Streak:         10,
		LastActiveDate: &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("users")
	col.InsertOne(ctx, user)

	streak, err := GetUserStreak(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if streak != 10 {
		t.Errorf("want streak=10 (same day, no change), got %d", streak)
	}
}

func TestGetUserStreak_UpdatesOnDiff(t *testing.T) {
	requireDB(t)
	clearCollection(t, "users")

	userID := primitive.NewObjectID()
	yesterday := time.Now().AddDate(0, 0, -1)
	user := models.User{
		ID:             userID,
		Name:           "Test",
		Phone:          "7777777777",
		Streak:         4,
		LastActiveDate: &yesterday,
		CreatedAt:      yesterday,
		UpdatedAt:      yesterday,
	}

	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("users")
	col.InsertOne(ctx, user)

	streak, err := GetUserStreak(ctx, userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if streak != 5 {
		t.Errorf("want streak=5 (consecutive day), got %d", streak)
	}
}

func TestGetUserStreak_NotFound(t *testing.T) {
	requireDB(t)
	clearCollection(t, "users")

	_, err := GetUserStreak(context.Background(), primitive.NewObjectID())
	if err == nil {
		t.Fatal("expected error for non-existent user")
	}
}
