package handlers

import (
	"context"
	"net/http"
	"regexp"
	"time"

	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

var phoneRegex = regexp.MustCompile(`^[6-9]\d{9}$`)

func SendOTP(c *gin.Context) {
	var body struct {
		Phone string `json:"phone" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "Phone number is required")
		return
	}
	if !phoneRegex.MatchString(body.Phone) {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_PHONE", "Enter a valid 10-digit Indian mobile number")
		return
	}

	err := utils.CreateOTP(body.Phone)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "OTP_SEND_FAILED", "Failed to send OTP")
		return
	}

	utils.Success(c, http.StatusOK, nil, "OTP sent successfully")
}

func VerifyOTP(c *gin.Context) {
	var body struct {
		Phone string `json:"phone" binding:"required"`
		OTP   string `json:"otp" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "Phone and OTP are required")
		return
	}

	valid, err := utils.VerifyOTP(body.Phone, body.OTP)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "VERIFY_FAILED", "Verification failed")
		return
	}
	if !valid {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_OTP", "OTP is incorrect or has expired")
		return
	}

	col := config.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user models.User
	err = col.FindOne(ctx, bson.M{"phone": body.Phone}).Decode(&user)

	if err == mongo.ErrNoDocuments {
		// New user — return temp token
		tempToken, err := utils.SignTempToken(body.Phone)
		if err != nil {
			utils.ErrorRes(c, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
			return
		}
		utils.Success(c, http.StatusOK, gin.H{
			"isNewUser": true,
			"tempToken": tempToken,
		}, "OTP verified. Please complete signup.")
		return
	}
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "DB_ERROR", "Database error")
		return
	}

	// Update streak and last active date on login
	newStreak := utils.CalculateStreak(user.LastActiveDate, user.Streak)
	now := time.Now()
	col.UpdateOne(ctx, bson.M{"_id": user.ID}, bson.M{
		"$set": bson.M{
			"lastActiveDate": now,
			"streak":         newStreak,
			"updatedAt":      now,
		},
	})

	// Fetch updated user data
	col.FindOne(ctx, bson.M{"_id": user.ID}).Decode(&user)

	// Existing user — return full token
	token, err := utils.SignToken(user.ID.Hex(), user.Phone)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}
	utils.Success(c, http.StatusOK, gin.H{
		"isNewUser": false,
		"token":     token,
		"user":      user,
	}, "Login successful")
}

func Signup(c *gin.Context) {
	phone, _ := c.Get("phone")

	var body struct {
		Name       string `json:"name" binding:"required"`
		ClassLevel string `json:"classLevel" binding:"required"`
		State      string `json:"state" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "Name, class level, and state are required")
		return
	}

	col := config.GetCollection("users")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if user already exists
	var existing models.User
	err := col.FindOne(ctx, bson.M{"phone": phone}).Decode(&existing)
	if err == nil {
		utils.ErrorRes(c, http.StatusBadRequest, "USER_EXISTS", "User already registered")
		return
	}

	now := time.Now()
	user := models.User{
		ID:             primitive.NewObjectID(),
		Name:           body.Name,
		Phone:          phone.(string),
		ClassLevel:     body.ClassLevel,
		State:          body.State,
		StarPoints:     0,
		Streak:         1, // First day streak
		IsPremium:      false,
		LastActiveDate: &now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	_, err = col.InsertOne(ctx, user)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "SIGNUP_FAILED", "Failed to create account")
		return
	}

	token, err := utils.SignToken(user.ID.Hex(), user.Phone)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}

	utils.Success(c, http.StatusCreated, gin.H{"token": token, "user": user}, "Account created successfully")
}
