package handlers

import (
	"context"
	"net/http"
	"time"

	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetSettings — GET /admin/settings
func GetSettings(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var settings models.Settings
	err := config.GetCollection("settings").FindOne(ctx, bson.M{}).Decode(&settings)
	if err != nil {
		// Return default settings if none exist
		settings = models.Settings{
			ExamName: "JNVST 2026",
			ExamDate: time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC), // Default to Nov 1, 2026
		}
	}

	utils.Success(c, http.StatusOK, gin.H{"settings": settings}, "Success")
}

// UpdateSettings — PUT /admin/settings (Super Admin only)
func UpdateSettings(c *gin.Context) {
	adminIDStr, _ := c.Get("adminId")
	adminID, _ := primitive.ObjectIDFromHex(adminIDStr.(string))

	var body struct {
		ExamDate string `json:"examDate" binding:"required"` // Format: "2026-11-01"
		ExamName string `json:"examName" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_INPUT", err.Error())
		return
	}

	// Parse exam date
	examDate, err := time.Parse("2006-01-02", body.ExamDate)
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_DATE", "Date must be in format YYYY-MM-DD")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Update or insert settings (upsert)
	filter := bson.M{}
	update := bson.M{
		"$set": bson.M{
			"examDate":  examDate,
			"examName":  body.ExamName,
			"updatedAt": time.Now(),
			"updatedBy": adminID,
		},
	}

	result, err := config.GetCollection("settings").UpdateOne(
		ctx, filter, update,
		nil,
	)

	// If no document exists, insert one
	if err != nil || result.MatchedCount == 0 {
		settings := models.Settings{
			ID:        primitive.NewObjectID(),
			ExamDate:  examDate,
			ExamName:  body.ExamName,
			UpdatedAt: time.Now(),
			UpdatedBy: adminID,
		}
		_, err = config.GetCollection("settings").InsertOne(ctx, settings)
		if err != nil {
			utils.ErrorRes(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update settings")
			return
		}
	}

	utils.Success(c, http.StatusOK, gin.H{
		"examDate": examDate,
		"examName": body.ExamName,
	}, "Settings updated successfully")
}
