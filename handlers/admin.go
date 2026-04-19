package handlers

import (
	"context"
	"log"
	"net/http"
	"time"

	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var test models.MockTest
	if err := config.GetCollection("mocktests").FindOne(ctx, bson.M{"_id": testID}).Decode(&test); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Mock test not found")
		return
	}

	questions := []models.Question{}
	if len(test.QuestionIDs) > 0 {
		qCtx, qCancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer qCancel()

		qCursor, err := config.GetCollection("questions").Find(qCtx, bson.M{"_id": bson.M{"$in": test.QuestionIDs}})
		if err != nil {
			log.Printf("ListAdminMockTestQuestions: Find error: %v", err)
			utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch questions: "+err.Error())
			return
		}
		defer qCursor.Close(qCtx)
		if err := qCursor.All(qCtx, &questions); err != nil {
			log.Printf("ListAdminMockTestQuestions: All error: %v", err)
			utils.ErrorRes(c, http.StatusInternalServerError, "DECODE_FAILED", "Failed to decode questions: "+err.Error())
			return
		}

		// MongoDB $in does not preserve insertion order — reorder to match test.QuestionIDs
		qMap := make(map[primitive.ObjectID]models.Question, len(questions))
		for _, q := range questions {
			qMap[q.ID] = q
		}
		ordered := make([]models.Question, 0, len(test.QuestionIDs))
		for _, id := range test.QuestionIDs {
			if q, ok := qMap[id]; ok {
				ordered = append(ordered, q)
			}
		}
		questions = ordered
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
// Body: { text, imageUrl, options[], correctIndex, explanation, subject, difficulty, classLevel, isPremium, tags[] }
// Creates the question and appends its ID to the test's questions array.
func AddQuestionToMockTest(c *gin.Context) {
	testID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid test ID")
		return
	}

	var body struct {
		Text         string                  `json:"text" binding:"required"`
		ImageURL     string                  `json:"imageUrl"`
		Options      []models.QuestionOption `json:"options" binding:"required"`
		CorrectIndex int                     `json:"correctIndex"`
		Explanation  string                  `json:"explanation"`
		Subject      string                  `json:"subject"`
		Difficulty   string                  `json:"difficulty"`
		ClassLevel   string                  `json:"classLevel"`
		IsPremium    bool                    `json:"isPremium"`
		IsPYQ        bool                    `json:"isPYQ"`
		ExamYear     string                  `json:"examYear"`
		Tags         []string                `json:"tags"`
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
		ImageURL:     body.ImageURL,
		Options:      body.Options,
		CorrectIndex: body.CorrectIndex,
		Explanation:  body.Explanation,
		Subject:      body.Subject,
		Difficulty:   body.Difficulty,
		ClassLevel:   body.ClassLevel,
		IsPremium:    body.IsPremium,
		IsPYQ:        body.IsPYQ,
		ExamYear:     body.ExamYear,
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

// DeleteMockTest — DELETE /admin/mocktests/:id
func DeleteMockTest(c *gin.Context) {
	testID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid test ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := config.GetCollection("mocktests").DeleteOne(ctx, bson.M{"_id": testID})
	if err != nil || result.DeletedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Mock test not found")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"deletedId": testID.Hex()}, "Mock test deleted")
}

// UpdateMockTestQuestion — PUT /admin/mocktests/:id/questions/:questionId
func UpdateMockTestQuestion(c *gin.Context) {
	questionID, err := primitive.ObjectIDFromHex(c.Param("questionId"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid question ID")
		return
	}

	var body struct {
		Text         string                  `json:"text" binding:"required"`
		ImageURL     string                  `json:"imageUrl"`
		Options      []models.QuestionOption `json:"options" binding:"required"`
		CorrectIndex int                     `json:"correctIndex"`
		Explanation  string                  `json:"explanation"`
		Subject      string                  `json:"subject"`
		Difficulty   string                  `json:"difficulty"`
		ClassLevel   string                  `json:"classLevel"`
		IsPremium    bool                    `json:"isPremium"`
		IsPYQ        bool                    `json:"isPYQ"`
		ExamYear     string                  `json:"examYear"`
		Tags         []string                `json:"tags"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "text and options are required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{
		"$set": bson.M{
			"text":         body.Text,
			"imageUrl":     body.ImageURL,
			"options":      body.Options,
			"correctIndex": body.CorrectIndex,
			"explanation":  body.Explanation,
			"subject":      body.Subject,
			"difficulty":   body.Difficulty,
			"classLevel":   body.ClassLevel,
			"isPremium":    body.IsPremium,
			"isPYQ":        body.IsPYQ,
			"examYear":     body.ExamYear,
			"tags":         body.Tags,
			"updatedAt":    time.Now(),
		},
	}

	result, err := config.GetCollection("questions").UpdateOne(ctx, bson.M{"_id": questionID}, update)
	if err != nil || result.MatchedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Question not found")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"questionId": questionID.Hex()}, "Question updated")
}

// DeleteMockTestQuestion — DELETE /admin/mocktests/:id/questions/:questionId
func DeleteMockTestQuestion(c *gin.Context) {
	testID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_TEST_ID", "Invalid test ID")
		return
	}

	questionID, err := primitive.ObjectIDFromHex(c.Param("questionId"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_QUESTION_ID", "Invalid question ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Remove question ID from test's questions array
	_, err = config.GetCollection("mocktests").UpdateOne(
		ctx,
		bson.M{"_id": testID},
		bson.M{"$pull": bson.M{"questions": questionID}},
	)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to remove question from test")
		return
	}

	// Delete the question document
	_, err = config.GetCollection("questions").DeleteOne(ctx, bson.M{"_id": questionID})
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete question")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"deletedId": questionID.Hex()}, "Question deleted")
}

// ReorderMockTestQuestions — PUT /admin/mocktests/:id/questions/reorder
// Body: { questionIds: [] } - array of question IDs in the new order
func ReorderMockTestQuestions(c *gin.Context) {
	testID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid test ID")
		return
	}

	var body struct {
		QuestionIDs []string `json:"questionIds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "questionIds array is required")
		return
	}

	// Convert string IDs to ObjectIDs
	questionObjIDs := make([]primitive.ObjectID, len(body.QuestionIDs))
	for i, idStr := range body.QuestionIDs {
		objID, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			utils.ErrorRes(c, http.StatusBadRequest, "INVALID_QUESTION_ID", "Invalid question ID in array")
			return
		}
		questionObjIDs[i] = objID
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Update the questions array in the test
	result, err := config.GetCollection("mocktests").UpdateOne(
		ctx,
		bson.M{"_id": testID},
		bson.M{"$set": bson.M{"questions": questionObjIDs}},
	)
	if err != nil || result.MatchedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Mock test not found")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"message": "Questions reordered successfully"}, "Success")
}
