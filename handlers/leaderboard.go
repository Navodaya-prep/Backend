package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"navodaya-api/config"
	"navodaya-api/utils"
)

func GetLeaderboard(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	weekAgo := time.Now().Add(-7 * 24 * time.Hour)

	pipeline := bson.A{
		bson.M{"$match": bson.M{"completedAt": bson.M{"$gte": weekAgo}}},
		bson.M{
			"$group": bson.M{
				"_id":        "$userId",
				"totalScore": bson.M{"$sum": "$score"},
				"testsCount": bson.M{"$sum": 1},
			},
		},
		bson.M{"$sort": bson.M{"totalScore": -1}},
		bson.M{"$limit": 50},
		bson.M{
			"$lookup": bson.M{
				"from":         "users",
				"localField":   "_id",
				"foreignField": "_id",
				"as":           "user",
			},
		},
		bson.M{"$unwind": "$user"},
		bson.M{
			"$project": bson.M{
				"_id":        1,
				"name":       "$user.name",
				"state":      "$user.state",
				"score":      "$totalScore",
				"testsCount": 1,
			},
		},
	}

	col := config.GetCollection("mocktestsattempts")
	cursor, err := col.Aggregate(ctx, pipeline)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch leaderboard")
		return
	}
	defer cursor.Close(ctx)

	var leaderboard []bson.M
	cursor.All(ctx, &leaderboard)
	if leaderboard == nil {
		leaderboard = []bson.M{}
	}

	// Find current user's rank
	userIDStr, _ := c.Get("userId")
	userRank := -1
	for i, entry := range leaderboard {
		if id, ok := entry["_id"]; ok {
			if id.(interface{ Hex() string }).Hex() == userIDStr.(string) {
				userRank = i + 1
				break
			}
		}
	}

	utils.Success(c, http.StatusOK, gin.H{
		"leaderboard": leaderboard,
		"userRank":    userRank,
	}, "Success")
}
