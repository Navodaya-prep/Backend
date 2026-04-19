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
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AddBookmark — POST /bookmarks
func AddBookmark(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	var body struct {
		QuestionID string `json:"questionId" binding:"required"`
		Source     string `json:"source"` // "practice" | "mocktest"
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "questionId is required")
		return
	}

	questionID, err := primitive.ObjectIDFromHex(body.QuestionID)
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid question ID")
		return
	}

	source := body.Source
	if source == "" {
		source = "practice"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Idempotent — don't duplicate
	existing, _ := config.GetCollection("bookmarks").CountDocuments(ctx, bson.M{
		"userId":     userID,
		"questionId": questionID,
	})
	if existing > 0 {
		utils.Success(c, http.StatusOK, nil, "Already bookmarked")
		return
	}

	bookmark := models.Bookmark{
		ID:         primitive.NewObjectID(),
		UserID:     userID,
		QuestionID: questionID,
		Source:     source,
		CreatedAt:  time.Now(),
	}

	if _, err := config.GetCollection("bookmarks").InsertOne(ctx, bookmark); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to bookmark question")
		return
	}
	utils.Success(c, http.StatusCreated, gin.H{"bookmark": bookmark}, "Question bookmarked")
}

// RemoveBookmark — DELETE /bookmarks/:questionId
func RemoveBookmark(c *gin.Context) {
	questionID, err := primitive.ObjectIDFromHex(c.Param("questionId"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid question ID")
		return
	}

	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("bookmarks").DeleteOne(ctx, bson.M{
		"userId":     userID,
		"questionId": questionID,
	})
	if err != nil || res.DeletedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Bookmark not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Bookmark removed")
}

// ListBookmarks — GET /bookmarks
func ListBookmarks(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pipeline := bson.A{
		bson.M{"$match": bson.M{"userId": userID}},
		bson.M{"$sort": bson.M{"createdAt": -1}},
		bson.M{
			"$lookup": bson.M{
				"from":         "questions",
				"localField":   "questionId",
				"foreignField": "_id",
				"as":           "question",
			},
		},
		bson.M{"$unwind": bson.M{"path": "$question", "preserveNullAndEmptyArrays": true}},
		bson.M{
			"$project": bson.M{
				"_id":              1,
				"source":           1,
				"createdAt":        1,
				"question._id":     1,
				"question.text":    1,
				"question.options": 1,
				"question.correctIndex": 1,
				"question.explanation":  1,
				"question.difficulty":   1,
				"question.subject":      1,
				"question.isPYQ":        1,
				"question.examYear":     1,
			},
		},
	}

	cursor, err := config.GetCollection("bookmarks").Aggregate(ctx, pipeline,
		options.Aggregate().SetMaxTime(10*time.Second))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch bookmarks")
		return
	}
	defer cursor.Close(ctx)

	var bookmarks []bson.M
	cursor.All(ctx, &bookmarks)
	if bookmarks == nil {
		bookmarks = []bson.M{}
	}

	utils.Success(c, http.StatusOK, gin.H{"bookmarks": bookmarks}, "Success")
}
