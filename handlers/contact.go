package handlers

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/navodayaprime/api/config"
	"github.com/navodayaprime/api/models"
	"github.com/navodayaprime/api/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var contactPhoneRegex = regexp.MustCompile(`^[6-9]\d{9}$`)

// SubmitContactMessage — POST /api/contact (public)
func SubmitContactMessage(c *gin.Context) {
	var body struct {
		Name    string `json:"name"`
		Phone   string `json:"phone"`
		Message string `json:"message"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_BODY", "Invalid request body")
		return
	}

	body.Name = strings.TrimSpace(body.Name)
	body.Phone = strings.TrimSpace(body.Phone)
	body.Message = strings.TrimSpace(body.Message)

	if body.Name == "" {
		utils.ErrorRes(c, http.StatusBadRequest, "VALIDATION_ERROR", "Name is required")
		return
	}
	if body.Phone == "" {
		utils.ErrorRes(c, http.StatusBadRequest, "VALIDATION_ERROR", "Phone number is required")
		return
	}
	if !contactPhoneRegex.MatchString(body.Phone) {
		utils.ErrorRes(c, http.StatusBadRequest, "VALIDATION_ERROR", "Enter a valid 10-digit Indian mobile number")
		return
	}
	if body.Message == "" {
		utils.ErrorRes(c, http.StatusBadRequest, "VALIDATION_ERROR", "Message is required")
		return
	}
	if len(body.Message) < 30 {
		utils.ErrorRes(c, http.StatusBadRequest, "VALIDATION_ERROR", "Message must be at least 30 characters")
		return
	}

	msg := models.ContactMessage{
		ID:        primitive.NewObjectID(),
		Name:      body.Name,
		Phone:     body.Phone,
		Message:   body.Message,
		CreatedAt: time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := config.GetCollection("contact_messages").InsertOne(ctx, msg); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "SAVE_FAILED", "Failed to save message, please try again")
		return
	}

	utils.Success(c, http.StatusOK, nil, "Message sent successfully")
}
