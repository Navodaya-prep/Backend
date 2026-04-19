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

// ListDoubts — GET /doubts?subject=&page=
func ListDoubts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{}
	if subject := c.Query("subject"); subject != "" {
		filter["subject"] = subject
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "createdAt", Value: -1}}).
		SetLimit(50)

	cursor, err := config.GetCollection("doubts").Find(ctx, filter, opts)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch doubts")
		return
	}
	defer cursor.Close(ctx)

	var doubts []models.Doubt
	cursor.All(ctx, &doubts)
	if doubts == nil {
		doubts = []models.Doubt{}
	}

	// Attach answer count to each doubt
	type DoubtWithCount struct {
		models.Doubt
		AnswerCount int `json:"answerCount"`
	}
	result := make([]DoubtWithCount, len(doubts))
	for i, d := range doubts {
		count, _ := config.GetCollection("doubtanswers").CountDocuments(ctx, bson.M{"doubtId": d.ID})
		result[i] = DoubtWithCount{Doubt: d, AnswerCount: int(count)}
	}

	utils.Success(c, http.StatusOK, gin.H{"doubts": result}, "Success")
}

// PostDoubt — POST /doubts
func PostDoubt(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	var body struct {
		Subject  string `json:"subject" binding:"required"`
		Text     string `json:"text" binding:"required"`
		ImageURL string `json:"imageUrl"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "subject and text are required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user struct {
		Name string `bson:"name"`
	}
	name := "Student"
	if err := config.GetCollection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err == nil {
		name = user.Name
	}

	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		UserName:  name,
		Subject:   body.Subject,
		Text:      body.Text,
		ImageURL:  body.ImageURL,
		Status:    "open",
		CreatedAt: time.Now(),
	}

	if _, err := config.GetCollection("doubts").InsertOne(ctx, doubt); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to post doubt")
		return
	}
	utils.Success(c, http.StatusCreated, gin.H{"doubt": doubt}, "Doubt posted")
}

// UpdateDoubt — PUT /doubts/:id (owner only, cannot change if already answered)
func UpdateDoubt(c *gin.Context) {
	doubtID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid doubt ID")
		return
	}

	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	var body struct {
		Subject string `json:"subject" binding:"required"`
		Text    string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "subject and text are required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("doubts").UpdateOne(ctx,
		bson.M{"_id": doubtID, "userId": userID},
		bson.M{"$set": bson.M{"subject": body.Subject, "text": body.Text}},
	)
	if err != nil || res.MatchedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Doubt not found or not yours")
		return
	}

	utils.Success(c, http.StatusOK, nil, "Doubt updated")
}

// GetDoubtAnswers — GET /doubts/:id/answers
func GetDoubtAnswers(c *gin.Context) {
	doubtID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid doubt ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var doubt models.Doubt
	if err := config.GetCollection("doubts").FindOne(ctx, bson.M{"_id": doubtID}).Decode(&doubt); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Doubt not found")
		return
	}

	cursor, err := config.GetCollection("doubtanswers").Find(ctx,
		bson.M{"doubtId": doubtID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch answers")
		return
	}
	defer cursor.Close(ctx)

	var answers []models.DoubtAnswer
	cursor.All(ctx, &answers)
	if answers == nil {
		answers = []models.DoubtAnswer{}
	}

	utils.Success(c, http.StatusOK, gin.H{"doubt": doubt, "answers": answers}, "Success")
}

// PostDoubtAnswer — POST /doubts/:id/answers
func PostDoubtAnswer(c *gin.Context) {
	doubtID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid doubt ID")
		return
	}

	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	var body struct {
		Text string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "text is required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify doubt exists
	count, _ := config.GetCollection("doubts").CountDocuments(ctx, bson.M{"_id": doubtID})
	if count == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Doubt not found")
		return
	}

	var u struct {
		Name string `bson:"name"`
	}
	name := "Student"
	if err := config.GetCollection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&u); err == nil {
		name = u.Name
	}

	answer := models.DoubtAnswer{
		ID:        primitive.NewObjectID(),
		DoubtID:   doubtID,
		UserID:    userID,
		UserName:  name,
		IsAdmin:   false,
		Text:      body.Text,
		CreatedAt: time.Now(),
	}

	if _, err := config.GetCollection("doubtanswers").InsertOne(ctx, answer); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to post answer")
		return
	}

	// Mark doubt as answered
	config.GetCollection("doubts").UpdateOne(ctx,
		bson.M{"_id": doubtID},
		bson.M{"$set": bson.M{"status": "answered"}})

	utils.Success(c, http.StatusCreated, gin.H{"answer": answer}, "Answer posted")
}

// DeleteDoubt — DELETE /doubts/:id (own doubts only)
func DeleteDoubt(c *gin.Context) {
	doubtID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid doubt ID")
		return
	}

	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("doubts").DeleteOne(ctx, bson.M{"_id": doubtID, "userId": userID})
	if err != nil || res.DeletedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Doubt not found or not yours")
		return
	}
	// Also delete associated answers
	config.GetCollection("doubtanswers").DeleteMany(ctx, bson.M{"doubtId": doubtID})

	utils.Success(c, http.StatusOK, nil, "Doubt deleted")
}

// ── Admin handlers ────────────────────────────────────────────────────────────

// AdminListDoubts — GET /admin/doubts
func AdminListDoubts(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{}
	if status := c.Query("status"); status != "" {
		filter["status"] = status
	}
	if subject := c.Query("subject"); subject != "" {
		filter["subject"] = subject
	}

	cursor, err := config.GetCollection("doubts").Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(100))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch doubts")
		return
	}
	defer cursor.Close(ctx)

	var doubts []models.Doubt
	cursor.All(ctx, &doubts)
	if doubts == nil {
		doubts = []models.Doubt{}
	}

	type DoubtWithCount struct {
		models.Doubt
		AnswerCount int `json:"answerCount"`
	}
	result := make([]DoubtWithCount, len(doubts))
	for i, d := range doubts {
		count, _ := config.GetCollection("doubtanswers").CountDocuments(ctx, bson.M{"doubtId": d.ID})
		result[i] = DoubtWithCount{Doubt: d, AnswerCount: int(count)}
	}

	utils.Success(c, http.StatusOK, gin.H{"doubts": result}, "Success")
}

// AdminAnswerDoubt — POST /admin/doubts/:id/answers
func AdminAnswerDoubt(c *gin.Context) {
	doubtID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid doubt ID")
		return
	}

	adminIDStr, _ := c.Get("adminId")
	adminID, _ := primitive.ObjectIDFromHex(adminIDStr.(string))
	adminEmail, _ := c.Get("adminEmail")

	var body struct {
		Text string `json:"text" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "text is required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	count, _ := config.GetCollection("doubts").CountDocuments(ctx, bson.M{"_id": doubtID})
	if count == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Doubt not found")
		return
	}

	name := "Admin"
	if adminEmail != nil {
		name = adminEmail.(string)
	}

	answer := models.DoubtAnswer{
		ID:        primitive.NewObjectID(),
		DoubtID:   doubtID,
		UserID:    adminID,
		UserName:  name,
		IsAdmin:   true,
		Text:      body.Text,
		CreatedAt: time.Now(),
	}

	if _, err := config.GetCollection("doubtanswers").InsertOne(ctx, answer); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to post answer")
		return
	}

	config.GetCollection("doubts").UpdateOne(ctx,
		bson.M{"_id": doubtID},
		bson.M{"$set": bson.M{"status": "answered"}})

	utils.Success(c, http.StatusCreated, gin.H{"answer": answer}, "Answer posted")
}

// AdminGetDoubtAnswers — GET /admin/doubts/:id/answers
func AdminGetDoubtAnswers(c *gin.Context) {
	doubtID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid doubt ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var doubt models.Doubt
	if err := config.GetCollection("doubts").FindOne(ctx, bson.M{"_id": doubtID}).Decode(&doubt); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Doubt not found")
		return
	}

	cursor, err := config.GetCollection("doubtanswers").Find(ctx,
		bson.M{"doubtId": doubtID},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: 1}}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch answers")
		return
	}
	defer cursor.Close(ctx)

	var answers []models.DoubtAnswer
	cursor.All(ctx, &answers)
	if answers == nil {
		answers = []models.DoubtAnswer{}
	}

	utils.Success(c, http.StatusOK, gin.H{"doubt": doubt, "answers": answers}, "Success")
}

// AdminDeleteDoubt — DELETE /admin/doubts/:id
func AdminDeleteDoubt(c *gin.Context) {
	doubtID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid doubt ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("doubts").DeleteOne(ctx, bson.M{"_id": doubtID})
	if err != nil || res.DeletedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Doubt not found")
		return
	}
	config.GetCollection("doubtanswers").DeleteMany(ctx, bson.M{"doubtId": doubtID})

	utils.Success(c, http.StatusOK, nil, "Doubt deleted")
}
