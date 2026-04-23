package handlers

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"
	"navodaya-api/ws"
)

// ─── Admin Handlers ──────────────────────────────────────────────────────────

// CreateLiveClass — POST /admin/live/classes
// Body: { title, subject, teacherName, description, classLevel, duration, isPremium }
func CreateLiveClass(c *gin.Context) {
	var body struct {
		Title       string `json:"title" binding:"required"`
		Subject     string `json:"subject" binding:"required"`
		TeacherName string `json:"teacherName" binding:"required"`
		Description string `json:"description"`
		ClassLevel  string `json:"classLevel" binding:"required"`
		Duration    int    `json:"duration" binding:"required"`
		IsPremium   bool   `json:"isPremium"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", err.Error())
		return
	}

	class := models.LiveClass{
		ID:          primitive.NewObjectID(),
		Title:       body.Title,
		Subject:     body.Subject,
		TeacherName: body.TeacherName,
		Description: body.Description,
		ClassLevel:  body.ClassLevel,
		Duration:    body.Duration,
		IsPremium:   body.IsPremium,
		IsLive:      true,
		StartedAt:   time.Now(),
		CreatedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := config.GetCollection("liveclasses").InsertOne(ctx, class); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create class")
		return
	}

	// Push notification to all users (non-blocking)
	utils.SendLiveClassNotification(class.ID.Hex(), class.Title, class.Subject)

	utils.Success(c, http.StatusCreated, gin.H{"class": class}, "Live class started")
}

// EndLiveClass — DELETE /admin/live/classes/:id
func EndLiveClass(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid class ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	now := time.Now()
	_, err = config.GetCollection("liveclasses").UpdateOne(ctx,
		bson.M{"_id": classID},
		bson.M{"$set": bson.M{"isLive": false, "endedAt": now}},
	)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to end class")
		return
	}

	// Notify all connected students
	ws.GlobalHub.BroadcastToStudents(classID.Hex(), ws.Message{Type: ws.EventClassEnd, Payload: nil})

	utils.Success(c, http.StatusOK, nil, "Live class ended")
}

// ListAdminLiveClasses — GET /admin/live/classes
func ListAdminLiveClasses(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	findOpts := options.Find().SetSort(bson.M{"createdAt": -1})
	cursor, err := config.GetCollection("liveclasses").Find(ctx, bson.M{}, findOpts)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch classes")
		return
	}
	defer cursor.Close(ctx)

	var classes []models.LiveClass
	cursor.All(ctx, &classes)
	if classes == nil {
		classes = []models.LiveClass{}
	}

	utils.Success(c, http.StatusOK, gin.H{"classes": classes}, "Success")
}

// PushLiveQuestion — POST /admin/live/classes/:id/questions
// Body: { text, options[], correctIndex, isReadOnly, timerSeconds }
func PushLiveQuestion(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid class ID")
		return
	}

	var body struct {
		Text         string   `json:"text" binding:"required"`
		Options      []string `json:"options"`
		CorrectIndex int      `json:"correctIndex"`
		IsReadOnly   bool     `json:"isReadOnly"`
		TimerSeconds int      `json:"timerSeconds" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", err.Error())
		return
	}
	if !body.IsReadOnly && len(body.Options) < 2 {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_OPTIONS", "At least 2 options required for answerable questions")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Deactivate any previously active question for this class
	config.GetCollection("livequestions").UpdateMany(ctx,
		bson.M{"liveClassId": classID, "isActive": true},
		bson.M{"$set": bson.M{"isActive": false}},
	)

	question := models.LiveQuestion{
		ID:           primitive.NewObjectID(),
		LiveClassID:  classID,
		Text:         body.Text,
		Options:      body.Options,
		CorrectIndex: body.CorrectIndex,
		IsReadOnly:   body.IsReadOnly,
		TimerSeconds: body.TimerSeconds,
		IsActive:     true,
		CreatedAt:    time.Now(),
	}
	if question.Options == nil {
		question.Options = []string{}
	}

	if _, err := config.GetCollection("livequestions").InsertOne(ctx, question); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to push question")
		return
	}

	// Broadcast quiz_start to all students
	ws.GlobalHub.BroadcastToStudents(classID.Hex(), ws.Message{
		Type: ws.EventQuizStart,
		Payload: ws.QuizStartPayload{
			QuestionID:   question.ID.Hex(),
			Text:         question.Text,
			Options:      question.Options,
			TimerSeconds: question.TimerSeconds,
			IsReadOnly:   question.IsReadOnly,
		},
	})

	utils.Success(c, http.StatusCreated, gin.H{"question": question}, "Question pushed")
}

// EndLiveQuestion — DELETE /admin/live/classes/:id/questions/:qid
func EndLiveQuestion(c *gin.Context) {
	questionID, err := primitive.ObjectIDFromHex(c.Param("qid"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid question ID")
		return
	}
	classIDStr := c.Param("id")
	classID, _ := primitive.ObjectIDFromHex(classIDStr)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var question models.LiveQuestion
	if err := config.GetCollection("livequestions").FindOne(ctx, bson.M{"_id": questionID}).Decode(&question); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Question not found")
		return
	}

	config.GetCollection("livequestions").UpdateOne(ctx,
		bson.M{"_id": questionID},
		bson.M{"$set": bson.M{"isActive": false}},
	)

	// Build leaderboard and broadcast quiz_end
	leaderboard := ws.GlobalHub.GetLeaderboard(questionID)
	ws.GlobalHub.BroadcastToClass(classID.Hex(), ws.Message{
		Type: ws.EventQuizEnd,
		Payload: ws.QuizEndPayload{
			QuestionID:   questionID.Hex(),
			CorrectIndex: question.CorrectIndex,
			Leaderboard:  leaderboard,
		},
	})

	utils.Success(c, http.StatusOK, gin.H{"leaderboard": leaderboard}, "Question ended")
}

// GetQuestionLeaderboard — GET /admin/live/classes/:id/questions/:qid/leaderboard
func GetQuestionLeaderboard(c *gin.Context) {
	questionID, err := primitive.ObjectIDFromHex(c.Param("qid"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid question ID")
		return
	}

	leaderboard := ws.GlobalHub.GetLeaderboard(questionID)
	utils.Success(c, http.StatusOK, gin.H{"leaderboard": leaderboard}, "Success")
}

// ─── Student Handlers ─────────────────────────────────────────────────────────

// ListActiveLiveClasses — GET /live/classes
func ListActiveLiveClasses(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	findOpts := options.Find().SetSort(bson.M{"startedAt": -1})
	cursor, err := config.GetCollection("liveclasses").Find(ctx, bson.M{"isLive": true}, findOpts)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch classes")
		return
	}
	defer cursor.Close(ctx)

	var classes []models.LiveClass
	cursor.All(ctx, &classes)
	if classes == nil {
		classes = []models.LiveClass{}
	}

	utils.Success(c, http.StatusOK, gin.H{"classes": classes}, "Success")
}

// GetLiveClass — GET /live/classes/:id
func GetLiveClass(c *gin.Context) {
	classID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid class ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var class models.LiveClass
	if err := config.GetCollection("liveclasses").FindOne(ctx, bson.M{"_id": classID}).Decode(&class); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Class not found")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"class": class}, "Success")
}

// GetAgoraToken — GET /live/classes/:id/agora-token  (student, audience role)
//               — GET /admin/live/classes/:id/agora-token  (teacher, publisher role)
func GetAgoraToken(c *gin.Context) {
	classIDStr := c.Param("id")
	if _, err := primitive.ObjectIDFromHex(classIDStr); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid class ID")
		return
	}

	// Determine role from context: admin middleware sets "isAdmin" key
	role := utils.AgoraRoleSubscriber
	if _, isAdmin := c.Get("isAdmin"); isAdmin {
		role = utils.AgoraRolePublisher
	}

	token := utils.BuildAgoraToken(classIDStr, "", role, 7200)

	utils.Success(c, http.StatusOK, gin.H{
		"token":       token,
		"appId":       getAgoraAppID(),
		"channelName": classIDStr,
		"uid":         0,
	}, "Token generated")
}

func getAgoraAppID() string {
	return os.Getenv("AGORA_APP_ID")
}

// RegisterPushToken — POST /users/push-token
// Body: { token, platform }
func RegisterPushToken(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	var body struct {
		Token    string `json:"token" binding:"required"`
		Platform string `json:"platform"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "token is required")
		return
	}
	if body.Platform == "" {
		body.Platform = "android"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	filter := bson.M{"userId": userID}
	update := bson.M{"$set": bson.M{
		"userId":    userID,
		"token":     body.Token,
		"platform":  body.Platform,
		"updatedAt": time.Now(),
	}, "$setOnInsert": bson.M{"_id": primitive.NewObjectID()}}

	opts := options.Update().SetUpsert(true)
	if _, err := config.GetCollection("pushtokens").UpdateOne(ctx, filter, update, opts); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "SAVE_FAILED", "Failed to save token")
		return
	}

	utils.Success(c, http.StatusOK, nil, "Push token registered")
}
