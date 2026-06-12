package handlers

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/navodayasarthi/api/config"
	"github.com/navodayasarthi/api/models"
	"github.com/navodayasarthi/api/utils"
)

var sendTimeRe = regexp.MustCompile(`^([01]\d|2[0-3]):[0-5]\d$`)

// ListNotificationTemplates — GET /admin/notifications/templates
// Seeds any missing defaults first, so new notification types appear automatically.
func ListNotificationTemplates(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	utils.SeedNotificationTemplates(ctx)

	cursor, err := config.GetCollection("notification_templates").Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "kind", Value: -1}, {Key: "name", Value: 1}}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch notification templates")
		return
	}
	defer cursor.Close(ctx)

	var templates []models.NotificationTemplate
	cursor.All(ctx, &templates)
	if templates == nil {
		templates = []models.NotificationTemplate{}
	}

	utils.Success(c, http.StatusOK, gin.H{"templates": templates}, "Success")
}

// UpdateNotificationTemplate — PUT /admin/notifications/templates/:key
// Admins edit the message content; key, kind and placeholders stay fixed.
func UpdateNotificationTemplate(c *gin.Context) {
	key := c.Param("key")

	var body struct {
		Title    *string `json:"title"`
		Body     *string `json:"body"`
		Enabled  *bool   `json:"enabled"`
		SendTime *string `json:"sendTime"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	set := bson.M{"updatedAt": time.Now()}
	if email, ok := c.Get("adminEmail"); ok {
		set["updatedBy"] = email
	}
	if body.Title != nil {
		if *body.Title == "" {
			utils.ErrorRes(c, http.StatusBadRequest, "INVALID_TITLE", "Title cannot be empty")
			return
		}
		set["title"] = *body.Title
	}
	if body.Body != nil {
		if *body.Body == "" {
			utils.ErrorRes(c, http.StatusBadRequest, "INVALID_BODY_TEXT", "Message body cannot be empty")
			return
		}
		set["body"] = *body.Body
	}
	if body.Enabled != nil {
		set["enabled"] = *body.Enabled
	}
	if body.SendTime != nil {
		if !sendTimeRe.MatchString(*body.SendTime) {
			utils.ErrorRes(c, http.StatusBadRequest, "INVALID_TIME", "Send time must be in HH:MM 24-hour format")
			return
		}
		set["sendTime"] = *body.SendTime
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("notification_templates").
		UpdateOne(ctx, bson.M{"key": key}, bson.M{"$set": set})
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "UPDATE_FAILED", "Failed to update template")
		return
	}
	if res.MatchedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Unknown notification template")
		return
	}

	var tpl models.NotificationTemplate
	config.GetCollection("notification_templates").FindOne(ctx, bson.M{"key": key}).Decode(&tpl)
	utils.Success(c, http.StatusOK, gin.H{"template": tpl}, "Template updated")
}

// BroadcastNotification — POST /admin/notifications/broadcast
// Sends a one-off announcement to every registered device, immediately.
func BroadcastNotification(c *gin.Context) {
	var body struct {
		Title string `json:"title" binding:"required"`
		Body  string `json:"body" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "title and body are required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sent := utils.SendPushToAll(ctx, body.Title, body.Body, map[string]string{"screen": "Home"})

	sentBy := "admin"
	if email, ok := c.Get("adminEmail"); ok {
		sentBy = email.(string)
	}
	utils.LogNotification(ctx, "broadcast", body.Title, body.Body, sentBy, sent)

	utils.Success(c, http.StatusOK, gin.H{"recipients": sent}, "Notification sent")
}

// ListNotificationLogs — GET /admin/notifications/logs
func ListNotificationLogs(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("notification_logs").Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "createdAt", Value: -1}}).SetLimit(30))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch notification history")
		return
	}
	defer cursor.Close(ctx)

	var logs []models.NotificationLog
	cursor.All(ctx, &logs)
	if logs == nil {
		logs = []models.NotificationLog{}
	}

	utils.Success(c, http.StatusOK, gin.H{"logs": logs}, "Success")
}
