package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"
)

// ListMockTests returns all tests with the current user's latest attempt attached
func ListMockTests(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Build subject/class filter
	matchFilter := bson.M{}
	if subject := c.Query("subject"); subject != "" {
		matchFilter["subject"] = subject
	}
	if class := c.Query("class"); class != "" {
		matchFilter["$or"] = bson.A{
			bson.M{"classLevel": class},
			bson.M{"classLevel": "both"},
		}
	}

	pipeline := bson.A{
		bson.M{"$match": matchFilter},
		bson.M{"$sort": bson.M{"createdAt": -1}},
		// Join with user's attempts for this test
		bson.M{
			"$lookup": bson.M{
				"from": "mocktestsattempts",
				"let":  bson.M{"testId": "$_id"},
				"pipeline": bson.A{
					bson.M{"$match": bson.M{"$expr": bson.M{"$and": bson.A{
						bson.M{"$eq": bson.A{"$mockTestId", "$$testId"}},
						bson.M{"$eq": bson.A{"$userId", userID}},
					}}}},
					bson.M{"$sort": bson.M{"completedAt": -1}},
					bson.M{"$limit": 1},
				},
				"as": "userAttempts",
			},
		},
		bson.M{
			"$addFields": bson.M{
				// Latest attempt or null
				"latestAttempt": bson.M{"$arrayElemAt": bson.A{"$userAttempts", 0}},
				// Dynamic question count
				"questionCount": bson.M{"$size": bson.M{"$ifNull": bson.A{"$questions", bson.A{}}}},
			},
		},
		// Remove heavy fields from list view
		bson.M{"$project": bson.M{
			"questions":    0,
			"userAttempts": 0,
		}},
	}

	cursor, err := config.GetCollection("mocktests").Aggregate(ctx, pipeline)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch tests")
		return
	}
	defer cursor.Close(ctx)

	var tests []bson.M
	cursor.All(ctx, &tests)
	if tests == nil {
		tests = []bson.M{}
	}

	utils.Success(c, http.StatusOK, gin.H{"tests": tests}, "Success")
}

// GetMockTest returns a full test with all questions populated
func GetMockTest(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid test ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var test models.MockTest
	if err := config.GetCollection("mocktests").FindOne(ctx, bson.M{"_id": id}).Decode(&test); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Test not found")
		return
	}

	// Populate questions dynamically
	if len(test.QuestionIDs) > 0 {
		qCursor, err := config.GetCollection("questions").Find(ctx, bson.M{"_id": bson.M{"$in": test.QuestionIDs}})
		if err == nil {
			qCursor.All(ctx, &test.Questions)
			qCursor.Close(ctx)
		}
	}

	utils.Success(c, http.StatusOK, gin.H{"test": test}, "Success")
}

// SubmitMockTest scores and saves a test attempt. Multiple attempts allowed.
func SubmitMockTest(c *gin.Context) {
	testID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid test ID")
		return
	}

	var body struct {
		// Keys are string question indices "0","1","2"...
		Answers   map[string]int `json:"answers" binding:"required"`
		TimeTaken int            `json:"timeTaken"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "Answers are required")
		return
	}

	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var test models.MockTest
	if err := config.GetCollection("mocktests").FindOne(ctx, bson.M{"_id": testID}).Decode(&test); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Test not found")
		return
	}

	// Fetch questions in the same order as stored
	qCursor, err := config.GetCollection("questions").Find(ctx, bson.M{"_id": bson.M{"$in": test.QuestionIDs}})
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch questions")
		return
	}
	defer qCursor.Close(ctx)

	var questions []models.Question
	qCursor.All(ctx, &questions)

	correct := 0
	attemptAnswers := make([]models.AttemptAnswer, len(questions))

	type DetailedResult struct {
		QuestionID   primitive.ObjectID `json:"questionId"`
		Text         string             `json:"text"`
		Options      []string           `json:"options"`
		SelectedIdx  int                `json:"selectedIndex"`
		CorrectIdx   int                `json:"correctIndex"`
		IsCorrect    bool               `json:"isCorrect"`
		Explanation  string             `json:"explanation"`
	}
	detailed := make([]DetailedResult, len(questions))

	for i, q := range questions {
		// Use strconv.Itoa so index 10+ works correctly ("10", "11", ...)
		key := strconv.Itoa(i)
		selectedIdx, exists := body.Answers[key]
		if !exists {
			selectedIdx = -1
		}
		isCorrect := selectedIdx == q.CorrectIndex
		if isCorrect {
			correct++
		}
		attemptAnswers[i] = models.AttemptAnswer{
			QuestionID:    q.ID,
			SelectedIndex: selectedIdx,
			IsCorrect:     isCorrect,
		}
		detailed[i] = DetailedResult{
			QuestionID:  q.ID,
			Text:        q.Text,
			Options:     q.Options,
			SelectedIdx: selectedIdx,
			CorrectIdx:  q.CorrectIndex,
			IsCorrect:   isCorrect,
			Explanation: q.Explanation,
		}
	}

	total := len(questions)
	percent := 0
	if total > 0 {
		percent = (correct * 100) / total
	}

	// Save attempt — always insert new so retest works
	attempt := models.MockTestAttempt{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		MockTestID:  testID,
		Answers:     attemptAnswers,
		Score:       correct,
		TotalMarks:  total,
		TimeTaken:   body.TimeTaken,
		CompletedAt: time.Now(),
	}
	config.GetCollection("mocktestsattempts").InsertOne(ctx, attempt)
	config.GetCollection("mocktests").UpdateOne(ctx, bson.M{"_id": testID}, bson.M{"$inc": bson.M{"attemptCount": 1}})

	utils.Success(c, http.StatusOK, gin.H{
		"result": gin.H{
			"attemptId":  attempt.ID,
			"score":      correct,
			"totalMarks": total,
			"correct":    correct,
			"wrong":      total - correct,
			"skipped":    total - len(body.Answers),
			"percent":    percent,
			"timeTaken":  body.TimeTaken,
			"detailed":   detailed,
		},
	}, "Test submitted successfully")
}

// GetUserAttempts returns all past attempts for the current user
func GetUserAttempts(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := bson.A{
		bson.M{"$match": bson.M{"userId": userID}},
		bson.M{"$sort": bson.M{"completedAt": -1}},
		bson.M{
			"$lookup": bson.M{
				"from":         "mocktests",
				"localField":   "mockTestId",
				"foreignField": "_id",
				"as":           "test",
			},
		},
		bson.M{"$unwind": "$test"},
		bson.M{
			"$project": bson.M{
				"score":       1,
				"totalMarks":  1,
				"timeTaken":   1,
				"completedAt": 1,
				"test.title":  1,
				"test.subject": 1,
				"test.duration": 1,
				"percent": bson.M{
					"$multiply": bson.A{
						bson.M{"$divide": bson.A{"$score", "$totalMarks"}},
						100,
					},
				},
			},
		},
	}

	cursor, err := config.GetCollection("mocktestsattempts").Aggregate(ctx, pipeline)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch attempts")
		return
	}
	defer cursor.Close(ctx)

	var attempts []bson.M
	cursor.All(ctx, &attempts)
	if attempts == nil {
		attempts = []bson.M{}
	}

	utils.Success(c, http.StatusOK, gin.H{"attempts": attempts}, "Success")
}
