package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"
)

func GetProfile(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	if err := config.GetCollection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}

	// Aggregate stats
	pipeline := bson.A{
		bson.M{"$match": bson.M{"userId": userID}},
		bson.M{
			"$group": bson.M{
				"_id":        nil,
				"totalTests": bson.M{"$sum": 1},
				"totalScore": bson.M{"$sum": "$score"},
				"bestScore":  bson.M{"$max": "$score"},
			},
		},
	}

	cursor, _ := config.GetCollection("mocktestsattempts").Aggregate(ctx, pipeline)
	var statsResult []bson.M
	cursor.All(ctx, &statsResult)

	stats := gin.H{"totalTests": 0, "totalScore": 0, "bestScore": 0}
	if len(statsResult) > 0 {
		stats = gin.H{
			"totalTests": statsResult[0]["totalTests"],
			"totalScore": statsResult[0]["totalScore"],
			"bestScore":  statsResult[0]["bestScore"],
		}
	}

	utils.Success(c, http.StatusOK, gin.H{"user": user, "stats": stats}, "Success")
}

func UpdateProfile(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	var body struct {
		Name  string `json:"name"`
		State string `json:"state"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "Invalid request body")
		return
	}

	update := bson.M{"$set": bson.M{"updatedAt": time.Now()}}
	if body.Name != "" {
		update["$set"].(bson.M)["name"] = body.Name
	}
	if body.State != "" {
		update["$set"].(bson.M)["state"] = body.State
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	config.GetCollection("users").UpdateOne(ctx, bson.M{"_id": userID}, update)

	var user models.User
	config.GetCollection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user)

	utils.Success(c, http.StatusOK, gin.H{"user": user}, "Profile updated")
}
