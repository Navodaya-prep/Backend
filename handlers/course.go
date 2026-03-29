package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"
)

func ListCourses(c *gin.Context) {
	subject := c.Query("subject")
	classLevel := c.Query("classLevel")

	filter := bson.M{}
	if subject != "" {
		filter["subject"] = subject
	}
	if classLevel != "" {
		filter["$or"] = bson.A{
			bson.M{"classLevel": classLevel},
			bson.M{"classLevel": "both"},
		}
	}

	col := config.GetCollection("courses")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.M{"order": 1})
	cursor, err := col.Find(ctx, filter, opts)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch courses")
		return
	}
	defer cursor.Close(ctx)

	var courses []models.Course
	if err := cursor.All(ctx, &courses); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "DECODE_FAILED", "Failed to decode courses")
		return
	}
	if courses == nil {
		courses = []models.Course{}
	}

	utils.Success(c, http.StatusOK, gin.H{"courses": courses}, "Success")
}

func GetCourse(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid course ID")
		return
	}

	col := config.GetCollection("courses")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var course models.Course
	if err := col.FindOne(ctx, bson.M{"_id": id}).Decode(&course); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Course not found")
		return
	}

	utils.Success(c, http.StatusOK, gin.H{"course": course}, "Success")
}

func GetCourseChapters(c *gin.Context) {
	courseID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid course ID")
		return
	}

	col := config.GetCollection("chapters")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.M{"order": 1})
	cursor, err := col.Find(ctx, bson.M{"courseId": courseID}, opts)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch chapters")
		return
	}
	defer cursor.Close(ctx)

	var chapters []models.Chapter
	if err := cursor.All(ctx, &chapters); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "DECODE_FAILED", "Failed to decode chapters")
		return
	}
	if chapters == nil {
		chapters = []models.Chapter{}
	}

	utils.Success(c, http.StatusOK, gin.H{"chapters": chapters}, "Success")
}
