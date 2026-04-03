package utils

import (
	"context"
	"time"

	"navodaya-api/config"
	"navodaya-api/models"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CalculateStreak calculates the current streak based on lastActiveDate
func CalculateStreak(lastActiveDate *time.Time, currentStreak int) int {
	if lastActiveDate == nil {
		return 1 // First activity
	}

	now := time.Now()
	lastActive := *lastActiveDate

	// Normalize times to start of day for comparison
	nowDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	lastDay := time.Date(lastActive.Year(), lastActive.Month(), lastActive.Day(), 0, 0, 0, 0, lastActive.Location())

	daysDiff := int(nowDay.Sub(lastDay).Hours() / 24)

	switch {
	case daysDiff == 0:
		// Same day - maintain current streak
		return currentStreak
	case daysDiff == 1:
		// Consecutive day - increment streak
		return currentStreak + 1
	default:
		// Streak broken - reset to 1
		return 1
	}
}

// UpdateUserActivity updates the user's last active date and streak
func UpdateUserActivity(ctx context.Context, userID primitive.ObjectID) error {
	col := config.GetCollection("users")

	// Get current user data
	var user models.User
	err := col.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return err
	}

	// Calculate new streak
	newStreak := CalculateStreak(user.LastActiveDate, user.Streak)
	now := time.Now()

	// Update user
	update := bson.M{
		"$set": bson.M{
			"lastActiveDate": now,
			"streak":         newStreak,
			"updatedAt":      now,
		},
	}

	_, err = col.UpdateOne(ctx, bson.M{"_id": userID}, update)
	return err
}

// GetUserStreak returns the current streak for a user
func GetUserStreak(ctx context.Context, userID primitive.ObjectID) (int, error) {
	col := config.GetCollection("users")

	var user models.User
	err := col.FindOne(ctx, bson.M{"_id": userID}).Decode(&user)
	if err != nil {
		return 0, err
	}

	// Recalculate streak based on last active date
	currentStreak := CalculateStreak(user.LastActiveDate, user.Streak)

	// If the calculated streak differs from stored, update it
	if currentStreak != user.Streak {
		now := time.Now()
		update := bson.M{
			"$set": bson.M{
				"streak":    currentStreak,
				"updatedAt": now,
			},
		}
		col.UpdateOne(ctx, bson.M{"_id": userID}, update)
	}

	return currentStreak, nil
}
