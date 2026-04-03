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
	"golang.org/x/crypto/bcrypt"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

// AdminLogin — POST /admin/auth/login
func AdminLogin(c *gin.Context) {
	var body struct {
		Email    string `json:"email" binding:"required"`
		Password string `json:"password" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "Email and password are required")
		return
	}

	if !emailRegex.MatchString(body.Email) {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_EMAIL", "Enter a valid email address")
		return
	}

	col := config.GetCollection("admins")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var admin models.Admin
	err := col.FindOne(ctx, bson.M{"email": body.Email}).Decode(&admin)
	if err == mongo.ErrNoDocuments {
		utils.ErrorRes(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "DB_ERROR", "Database error")
		return
	}

	if !admin.IsActive {
		utils.ErrorRes(c, http.StatusForbidden, "ACCOUNT_INACTIVE", "Your admin account has been deactivated")
		return
	}

	// Verify password
	err = bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(body.Password))
	if err != nil {
		utils.ErrorRes(c, http.StatusUnauthorized, "INVALID_CREDENTIALS", "Invalid email or password")
		return
	}

	// Generate JWT token for admin
	token, err := utils.SignAdminToken(admin.ID.Hex(), admin.Email, admin.IsSuperAdmin)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "TOKEN_ERROR", "Failed to generate token")
		return
	}

	// Update last login
	col.UpdateOne(ctx, bson.M{"_id": admin.ID}, bson.M{
		"$set": bson.M{"updatedAt": time.Now()},
	})

	utils.Success(c, http.StatusOK, gin.H{
		"token": token,
		"admin": admin,
	}, "Login successful")
}

// GetAdminProfile — GET /admin/auth/profile
func GetAdminProfile(c *gin.Context) {
	adminIDStr, _ := c.Get("adminId")
	adminID, err := primitive.ObjectIDFromHex(adminIDStr.(string))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid admin ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var admin models.Admin
	if err := config.GetCollection("admins").FindOne(ctx, bson.M{"_id": adminID}).Decode(&admin); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Admin not found")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"admin": admin}, "Success")
}

// UpdateAdminProfile — PUT /admin/auth/profile
func UpdateAdminProfile(c *gin.Context) {
	adminIDStr, _ := c.Get("adminId")
	adminID, _ := primitive.ObjectIDFromHex(adminIDStr.(string))

	var body struct {
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Email     string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	// Validate email if provided
	if body.Email != "" && !emailRegex.MatchString(body.Email) {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_EMAIL", "Enter a valid email address")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{"updatedAt": time.Now()}}
	if body.FirstName != "" {
		update["$set"].(bson.M)["firstName"] = body.FirstName
	}
	if body.LastName != "" {
		update["$set"].(bson.M)["lastName"] = body.LastName
	}
	if body.Email != "" {
		// Check if email already exists
		var existingAdmin models.Admin
		err := config.GetCollection("admins").FindOne(ctx, bson.M{"email": body.Email, "_id": bson.M{"$ne": adminID}}).Decode(&existingAdmin)
		if err == nil {
			utils.ErrorRes(c, http.StatusConflict, "EMAIL_EXISTS", "Email already in use")
			return
		}
		update["$set"].(bson.M)["email"] = body.Email
	}

	config.GetCollection("admins").UpdateOne(ctx, bson.M{"_id": adminID}, update)

	var admin models.Admin
	config.GetCollection("admins").FindOne(ctx, bson.M{"_id": adminID}).Decode(&admin)

	utils.Success(c, http.StatusOK, gin.H{"admin": admin}, "Profile updated")
}

// ChangeAdminPassword — PUT /admin/auth/change-password
func ChangeAdminPassword(c *gin.Context) {
	adminIDStr, _ := c.Get("adminId")
	adminID, _ := primitive.ObjectIDFromHex(adminIDStr.(string))

	var body struct {
		CurrentPassword string `json:"currentPassword" binding:"required"`
		NewPassword     string `json:"newPassword" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "Current and new password required")
		return
	}

	if len(body.NewPassword) < 6 {
		utils.ErrorRes(c, http.StatusBadRequest, "WEAK_PASSWORD", "Password must be at least 6 characters")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var admin models.Admin
	if err := config.GetCollection("admins").FindOne(ctx, bson.M{"_id": adminID}).Decode(&admin); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Admin not found")
		return
	}

	// Verify current password
	if err := bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(body.CurrentPassword)); err != nil {
		utils.ErrorRes(c, http.StatusUnauthorized, "INVALID_PASSWORD", "Current password is incorrect")
		return
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "HASH_ERROR", "Failed to hash password")
		return
	}

	// Update password
	config.GetCollection("admins").UpdateOne(ctx, bson.M{"_id": adminID}, bson.M{
		"$set": bson.M{"password": string(hashedPassword), "updatedAt": time.Now()},
	})

	utils.Success(c, http.StatusOK, nil, "Password changed successfully")
}
