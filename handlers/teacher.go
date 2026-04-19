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
	"golang.org/x/crypto/bcrypt"
)

// ListTeachers — GET /admin/manage/teachers
func ListTeachers(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("teachers").Find(ctx, bson.M{})
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "DB_ERROR", "Failed to fetch teachers")
		return
	}
	defer cursor.Close(ctx)

	var teachers []models.Teacher
	if err := cursor.All(ctx, &teachers); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "DB_ERROR", "Failed to decode teachers")
		return
	}
	if teachers == nil {
		teachers = []models.Teacher{}
	}

	utils.Success(c, http.StatusOK, gin.H{"teachers": teachers}, "Success")
}

// InviteTeacher — POST /admin/manage/teachers/invite
func InviteTeacher(c *gin.Context) {
	var body struct {
		FirstName  string `json:"firstName" binding:"required"`
		LastName   string `json:"lastName" binding:"required"`
		Email      string `json:"email" binding:"required"`
		Phone      string `json:"phone"`
		Subject    string `json:"subject"`
		ClassLevel string `json:"classLevel"`
		Bio        string `json:"bio"`
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

	var existing models.Teacher
	if err := config.GetCollection("teachers").FindOne(ctx, bson.M{"email": body.Email}).Decode(&existing); err == nil {
		utils.ErrorRes(c, http.StatusConflict, "EMAIL_EXISTS", "A teacher with this email already exists")
		return
	}

	tempPassword := generateRandomPassword(12)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "HASH_ERROR", "Failed to hash password")
		return
	}

	teacher := models.Teacher{
		ID:         primitive.NewObjectID(),
		FirstName:  body.FirstName,
		LastName:   body.LastName,
		Email:      body.Email,
		Password:   string(hashedPassword),
		Phone:      body.Phone,
		Subject:    body.Subject,
		ClassLevel: body.ClassLevel,
		Bio:        body.Bio,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	if _, err := config.GetCollection("teachers").InsertOne(ctx, teacher); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "INSERT_ERROR", "Failed to create teacher")
		return
	}

	utils.Success(c, http.StatusCreated, gin.H{
		"teacher":      teacher,
		"tempPassword": tempPassword,
	}, "Teacher invited successfully")
}

// UpdateTeacher — PUT /admin/manage/teachers/:id
func UpdateTeacher(c *gin.Context) {
	teacherID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid teacher ID")
		return
	}

	var body struct {
		FirstName  string `json:"firstName" binding:"required"`
		LastName   string `json:"lastName" binding:"required"`
		Phone      string `json:"phone"`
		Subject    string `json:"subject"`
		ClassLevel string `json:"classLevel"`
		Bio        string `json:"bio"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "First name and last name are required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := config.GetCollection("teachers").UpdateOne(ctx, bson.M{"_id": teacherID}, bson.M{
		"$set": bson.M{
			"firstName":  body.FirstName,
			"lastName":   body.LastName,
			"phone":      body.Phone,
			"subject":    body.Subject,
			"classLevel": body.ClassLevel,
			"bio":        body.Bio,
			"updatedAt":  time.Now(),
		},
	})
	if err != nil || result.MatchedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Teacher not found")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"id": teacherID.Hex()}, "Teacher updated successfully")
}

// ToggleTeacherStatus — PUT /admin/manage/teachers/:id/toggle
func ToggleTeacherStatus(c *gin.Context) {
	teacherID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid teacher ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var teacher models.Teacher
	if err := config.GetCollection("teachers").FindOne(ctx, bson.M{"_id": teacherID}).Decode(&teacher); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Teacher not found")
		return
	}

	newStatus := !teacher.IsActive
	config.GetCollection("teachers").UpdateOne(ctx, bson.M{"_id": teacherID}, bson.M{
		"$set": bson.M{"isActive": newStatus, "updatedAt": time.Now()},
	})

	utils.Success(c, http.StatusOK, gin.H{"isActive": newStatus}, "Status updated")
}

// DeleteTeacher — DELETE /admin/manage/teachers/:id
func DeleteTeacher(c *gin.Context) {
	teacherID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid teacher ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result, err := config.GetCollection("teachers").DeleteOne(ctx, bson.M{"_id": teacherID})
	if err != nil || result.DeletedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Teacher not found")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"deletedId": teacherID.Hex()}, "Teacher deleted successfully")
}
