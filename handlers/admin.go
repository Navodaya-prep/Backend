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

// ListAdminMockTests — GET /admin/mocktests
func ListAdminMockTests(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pipeline := bson.A{
		bson.M{"$sort": bson.M{"createdAt": -1}},
		bson.M{"$addFields": bson.M{
			"questionCount": bson.M{"$size": bson.M{"$ifNull": bson.A{"$questions", bson.A{}}}},
		}},
		bson.M{"$project": bson.M{"questions": 0}},
	}

	cursor, err := config.GetCollection("mocktests").Aggregate(ctx, pipeline)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch mock tests")
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

// ListAdminMockTestQuestions — GET /admin/mocktests/:id/questions
func ListAdminMockTestQuestions(c *gin.Context) {
	testID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid test ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var test models.MockTest
	if err := config.GetCollection("mocktests").FindOne(ctx, bson.M{"_id": testID}).Decode(&test); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Mock test not found")
		return
	}

	questions := []models.Question{}
	if len(test.QuestionIDs) > 0 {
		qCursor, err := config.GetCollection("questions").Find(ctx, bson.M{"_id": bson.M{"$in": test.QuestionIDs}})
		if err == nil {
			qCursor.All(ctx, &questions)
			qCursor.Close(ctx)
		}
	}

	utils.Success(c, http.StatusOK, gin.H{"questions": questions}, "Success")
}

// CreateMockTest — POST /admin/mocktests
// Body: { title, subject, duration, classLevel, isPremium }
func CreateMockTest(c *gin.Context) {
	var body struct {
		Title      string `json:"title" binding:"required"`
		Subject    string `json:"subject" binding:"required"`
		Duration   int    `json:"duration" binding:"required"` // minutes
		ClassLevel string `json:"classLevel" binding:"required"`
		IsPremium  bool   `json:"isPremium"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "title, subject, duration, classLevel are required")
		return
	}

	test := models.MockTest{
		ID:          primitive.NewObjectID(),
		Title:       body.Title,
		Subject:     body.Subject,
		Duration:    body.Duration,
		ClassLevel:  body.ClassLevel,
		IsPremium:   body.IsPremium,
		QuestionIDs: []primitive.ObjectID{},
		CreatedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := config.GetCollection("mocktests").InsertOne(ctx, test); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create mock test")
		return
	}

	utils.Success(c, http.StatusCreated, gin.H{"test": test}, "Mock test created")
}

// AddQuestionToMockTest — POST /admin/mocktests/:id/questions
// Body: { text, options[], correctIndex, explanation, subject, difficulty, classLevel, isPremium, tags[] }
// Creates the question and appends its ID to the test's questions array.
func AddQuestionToMockTest(c *gin.Context) {
	testID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid test ID")
		return
	}

	var body struct {
		Text         string   `json:"text" binding:"required"`
		Options      []string `json:"options" binding:"required"`
		CorrectIndex int      `json:"correctIndex"`
		Explanation  string   `json:"explanation"`
		Subject      string   `json:"subject"`
		Difficulty   string   `json:"difficulty"`
		ClassLevel   string   `json:"classLevel"`
		IsPremium    bool     `json:"isPremium"`
		Tags         []string `json:"tags"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "text and options are required")
		return
	}
	if len(body.Options) < 2 {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_OPTIONS", "At least 2 options are required")
		return
	}
	if body.CorrectIndex < 0 || body.CorrectIndex >= len(body.Options) {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_CORRECT_INDEX", "correctIndex out of range")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify test exists
	count, err := config.GetCollection("mocktests").CountDocuments(ctx, bson.M{"_id": testID})
	if err != nil || count == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Mock test not found")
		return
	}

	question := models.Question{
		ID:           primitive.NewObjectID(),
		Text:         body.Text,
		Options:      body.Options,
		CorrectIndex: body.CorrectIndex,
		Explanation:  body.Explanation,
		Subject:      body.Subject,
		Difficulty:   body.Difficulty,
		ClassLevel:   body.ClassLevel,
		IsPremium:    body.IsPremium,
		Tags:         body.Tags,
		CreatedAt:    time.Now(),
	}
	if question.Tags == nil {
		question.Tags = []string{}
	}

	// Insert question
	if _, err := config.GetCollection("questions").InsertOne(ctx, question); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to save question")
		return
	}

	// Append question ID to the test
	if _, err := config.GetCollection("mocktests").UpdateOne(
		ctx,
		bson.M{"_id": testID},
		bson.M{"$push": bson.M{"questions": question.ID}},
	); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "UPDATE_FAILED", "Question saved but failed to link to test")
		return
	}

	utils.Success(c, http.StatusCreated, gin.H{"question": question}, "Question added to test")
}
