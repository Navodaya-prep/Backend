package handlers

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"time"

	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// ListAdmins — GET /admin/manage/admins (Super Admin only)
func ListAdmins(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("admins").Find(ctx, bson.M{})
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch admins")
		return
	}
	defer cursor.Close(ctx)

	var admins []models.Admin
	if err := cursor.All(ctx, &admins); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "DB_ERROR", "Failed to decode admins")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"admins": admins}, "Success")
}

// DeleteAdmin — DELETE /admin/manage/admins/:id (Super Admin only)
func DeleteAdmin(c *gin.Context) {
	targetAdminID := c.Param("id")
	currentAdminIDStr, _ := c.Get("adminId")

	// Prevent deleting self
	if targetAdminID == currentAdminIDStr.(string) {
		utils.ErrorRes(c, http.StatusBadRequest, "CANNOT_DELETE_SELF", "You cannot delete your own account")
		return
	}

	targetObjID, err := primitive.ObjectIDFromHex(targetAdminID)
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid admin ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if admin exists and is not the last super admin
	var targetAdmin models.Admin
	if err := config.GetCollection("admins").FindOne(ctx, bson.M{"_id": targetObjID}).Decode(&targetAdmin); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Admin not found")
		return
	}

	// If deleting a super admin, ensure at least one super admin will remain
	if targetAdmin.IsSuperAdmin {
		superAdminCount, _ := config.GetCollection("admins").CountDocuments(ctx, bson.M{"isSuperAdmin": true})
		if superAdminCount <= 1 {
			utils.ErrorRes(c, http.StatusBadRequest, "LAST_SUPER_ADMIN", "Cannot delete the last super admin")
			return
		}
	}

	result, err := config.GetCollection("admins").DeleteOne(ctx, bson.M{"_id": targetObjID})
	if err != nil || result.DeletedCount == 0 {
		utils.ErrorRes(c, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete admin")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"deletedId": targetAdminID}, "Admin deleted successfully")
}

// InviteAdmin — POST /admin/manage/admins/invite (Super Admin only)
func InviteAdmin(c *gin.Context) {
	var body struct {
		FirstName    string `json:"firstName" binding:"required"`
		LastName     string `json:"lastName" binding:"required"`
		Email        string `json:"email" binding:"required"`
		IsSuperAdmin bool   `json:"isSuperAdmin"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "First name, last name, and email are required")
		return
	}

	if !emailRegex.MatchString(body.Email) {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_EMAIL", "Enter a valid email address")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if email already exists
	var existingAdmin models.Admin
	err := config.GetCollection("admins").FindOne(ctx, bson.M{"email": body.Email}).Decode(&existingAdmin)
	if err == nil {
		utils.ErrorRes(c, http.StatusConflict, "EMAIL_EXISTS", "An admin with this email already exists")
		return
	}

	// Generate a random temporary password
	tempPassword := generateRandomPassword(12)

	// Hash the temporary password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "HASH_ERROR", "Failed to hash password")
		return
	}

	// Create new admin
	newAdmin := models.Admin{
		ID:           primitive.NewObjectID(),
		FirstName:    body.FirstName,
		LastName:     body.LastName,
		Email:        body.Email,
		Password:     string(hashedPassword),
		IsSuperAdmin: body.IsSuperAdmin,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	_, err = config.GetCollection("admins").InsertOne(ctx, newAdmin)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "INSERT_ERROR", "Failed to create admin")
		return
	}

	// TODO: Send email with temporary password
	// For now, we return the password in the response
	// In production, you should send this via email and NOT return it in the response

	utils.Success(c, http.StatusCreated, gin.H{
		"admin":         newAdmin,
		"tempPassword":  tempPassword,
		"message":       "Admin invited. Temporary password generated. Please send this password via email.",
		"emailRequired": true,
	}, "Admin created successfully")
}

// generateRandomPassword creates a secure random password
func generateRandomPassword(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}
