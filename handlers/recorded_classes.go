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

// ─── Admin: Course CRUD ───────────────────────────────────────────────────────

// AdminListCourses — GET /admin/courses
func AdminListCourses(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("courses").Find(ctx, bson.M{},
		options.Find().SetSort(bson.M{"order": 1}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch courses")
		return
	}
	defer cursor.Close(ctx)

	var courses []models.Course
	cursor.All(ctx, &courses)
	if courses == nil {
		courses = []models.Course{}
	}
	utils.Success(c, http.StatusOK, gin.H{"courses": courses}, "Success")
}

// AdminCreateCourse — POST /admin/courses
func AdminCreateCourse(c *gin.Context) {
	var body struct {
		Title       string `json:"title" binding:"required"`
		Subject     string `json:"subject" binding:"required"`
		ClassLevel  string `json:"classLevel"`
		Thumbnail   string `json:"thumbnail"`
		Description string `json:"description"`
		Order       int    `json:"order"`
		IsPremium   bool   `json:"isPremium"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", err.Error())
		return
	}

	course := models.Course{
		ID:          primitive.NewObjectID(),
		Title:       body.Title,
		Subject:     body.Subject,
		ClassLevel:  body.ClassLevel,
		Thumbnail:   body.Thumbnail,
		Description: body.Description,
		Order:       body.Order,
		IsPremium:   body.IsPremium,
		CreatedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := config.GetCollection("courses").InsertOne(ctx, course); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create course")
		return
	}
	utils.Success(c, http.StatusCreated, gin.H{"course": course}, "Course created")
}

// AdminUpdateCourse — PUT /admin/courses/:id
func AdminUpdateCourse(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid course ID")
		return
	}

	var body struct {
		Title       string `json:"title"`
		Subject     string `json:"subject"`
		ClassLevel  string `json:"classLevel"`
		Thumbnail   string `json:"thumbnail"`
		Description string `json:"description"`
		Order       int    `json:"order"`
		IsPremium   bool   `json:"isPremium"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("courses").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"title": body.Title, "subject": body.Subject, "classLevel": body.ClassLevel,
		"thumbnail": body.Thumbnail, "description": body.Description,
		"order": body.Order, "isPremium": body.IsPremium,
	}})
	if err != nil || res.MatchedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Course not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Course updated")
}

// AdminDeleteCourse — DELETE /admin/courses/:id
func AdminDeleteCourse(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid course ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("courses").DeleteOne(ctx, bson.M{"_id": id})
	if err != nil || res.DeletedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Course not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Course deleted")
}

// ─── Admin: Course Chapter CRUD ───────────────────────────────────────────────

// AdminListCourseChapters — GET /admin/courses/:id/chapters
func AdminListCourseChapters(c *gin.Context) {
	courseID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid course ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("chapters").Find(ctx,
		bson.M{"courseId": courseID},
		options.Find().SetSort(bson.M{"order": 1}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch chapters")
		return
	}
	defer cursor.Close(ctx)

	var chapters []models.Chapter
	cursor.All(ctx, &chapters)
	if chapters == nil {
		chapters = []models.Chapter{}
	}
	utils.Success(c, http.StatusOK, gin.H{"chapters": chapters}, "Success")
}

// AdminCreateCourseChapter — POST /admin/courses/:id/chapters
func AdminCreateCourseChapter(c *gin.Context) {
	courseID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid course ID")
		return
	}

	var body struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		Order       int    `json:"order"`
		IsPremium   bool   `json:"isPremium"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", err.Error())
		return
	}

	chapter := models.Chapter{
		ID:          primitive.NewObjectID(),
		CourseID:    &courseID,
		Title:       body.Title,
		Description: body.Description,
		Order:       body.Order,
		IsPremium:   body.IsPremium,
		CreatedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := config.GetCollection("chapters").InsertOne(ctx, chapter); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create chapter")
		return
	}

	// Update course chapter count
	config.GetCollection("courses").UpdateOne(ctx,
		bson.M{"_id": courseID},
		bson.M{"$inc": bson.M{"chaptersCount": 1}},
	)

	utils.Success(c, http.StatusCreated, gin.H{"chapter": chapter}, "Chapter created")
}

// AdminUpdateCourseChapter — PUT /admin/chapters/:id
// (Reuses AdminUpdateChapter from practice_hub.go since model is shared)

// AdminDeleteCourseChapter — DELETE /admin/chapters/:id
// (Reuses AdminDeleteChapter from practice_hub.go since model is shared)

// ─── Admin: Lesson CRUD ───────────────────────────────────────────────────────

// AdminListLessons — GET /admin/chapters/:id/lessons
func AdminListLessons(c *gin.Context) {
	chapterID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid chapter ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("lessons").Find(ctx,
		bson.M{"chapterId": chapterID},
		options.Find().SetSort(bson.M{"order": 1}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch lessons")
		return
	}
	defer cursor.Close(ctx)

	var lessons []models.Lesson
	cursor.All(ctx, &lessons)
	if lessons == nil {
		lessons = []models.Lesson{}
	}
	utils.Success(c, http.StatusOK, gin.H{"lessons": lessons}, "Success")
}

// AdminCreateLesson — POST /admin/chapters/:id/lessons
func AdminCreateLesson(c *gin.Context) {
	chapterID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid chapter ID")
		return
	}

	var body struct {
		Title          string `json:"title" binding:"required"`
		Type           string `json:"type" binding:"required"` // "video" | "note"
		YouTubeVideoID string `json:"youtubeVideoId"`
		NoteContent    string `json:"noteContent"`
		Description    string `json:"description"`
		DurationMins   int    `json:"durationMins"`
		Order          int    `json:"order"`
		IsPremium      bool   `json:"isPremium"`
		CourseID       string `json:"courseId" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", err.Error())
		return
	}
	if body.Type != "video" && body.Type != "note" {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_TYPE", "type must be 'video' or 'note'")
		return
	}

	courseID, err := primitive.ObjectIDFromHex(body.CourseID)
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid course ID")
		return
	}

	lesson := models.Lesson{
		ID:             primitive.NewObjectID(),
		ChapterID:      chapterID,
		CourseID:       courseID,
		Title:          body.Title,
		Type:           body.Type,
		YouTubeVideoID: body.YouTubeVideoID,
		NoteContent:    body.NoteContent,
		Description:    body.Description,
		DurationMins:   body.DurationMins,
		Order:          body.Order,
		IsPremium:      body.IsPremium,
		CreatedAt:      time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := config.GetCollection("lessons").InsertOne(ctx, lesson); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create lesson")
		return
	}

	// Update course videos count if it's a video lesson
	if body.Type == "video" {
		config.GetCollection("courses").UpdateOne(ctx,
			bson.M{"_id": courseID},
			bson.M{"$inc": bson.M{"videosCount": 1}},
		)
	}

	utils.Success(c, http.StatusCreated, gin.H{"lesson": lesson}, "Lesson created")
}

// AdminUpdateLesson — PUT /admin/lessons/:id
func AdminUpdateLesson(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid lesson ID")
		return
	}

	var body struct {
		Title          string `json:"title"`
		Type           string `json:"type"`
		YouTubeVideoID string `json:"youtubeVideoId"`
		NoteContent    string `json:"noteContent"`
		Description    string `json:"description"`
		DurationMins   int    `json:"durationMins"`
		Order          int    `json:"order"`
		IsPremium      bool   `json:"isPremium"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("lessons").UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": bson.M{
		"title": body.Title, "type": body.Type, "youtubeVideoId": body.YouTubeVideoID,
		"noteContent": body.NoteContent, "description": body.Description,
		"durationMins": body.DurationMins, "order": body.Order, "isPremium": body.IsPremium,
	}})
	if err != nil || res.MatchedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Lesson not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Lesson updated")
}

// AdminDeleteLesson — DELETE /admin/lessons/:id
func AdminDeleteLesson(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid lesson ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("lessons").DeleteOne(ctx, bson.M{"_id": id})
	if err != nil || res.DeletedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Lesson not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Lesson deleted")
}

// ─── Student: Recorded Classes ────────────────────────────────────────────────

// GetCourseChaptersWithProgress — GET /courses/:id/chapters/progress
// Returns chapters enriched with lesson count and user's completed count.
func GetCourseChaptersWithProgress(c *gin.Context) {
	courseID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid course ID")
		return
	}
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("chapters").Find(ctx,
		bson.M{"courseId": courseID},
		options.Find().SetSort(bson.M{"order": 1}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch chapters")
		return
	}
	defer cursor.Close(ctx)

	var chapters []models.Chapter
	cursor.All(ctx, &chapters)
	if chapters == nil {
		chapters = []models.Chapter{}
	}

	// Fetch user's overall progress for this course
	var progress models.UserCourseProgress
	config.GetCollection("usercourseprogress").FindOne(ctx,
		bson.M{"userId": userID, "courseId": courseID}).Decode(&progress)

	completedSet := make(map[primitive.ObjectID]bool)
	for _, id := range progress.CompletedLessonIDs {
		completedSet[id] = true
	}

	type ChapterWithProgress struct {
		models.Chapter
		LessonCount     int `json:"lessonCount"`
		CompletedCount  int `json:"completedCount"`
	}
	result := make([]ChapterWithProgress, len(chapters))
	for i, ch := range chapters {
		lessonCursor, _ := config.GetCollection("lessons").Find(ctx,
			bson.M{"chapterId": ch.ID})
		var lessons []models.Lesson
		lessonCursor.All(ctx, &lessons)
		lessonCursor.Close(ctx)

		completed := 0
		for _, l := range lessons {
			if completedSet[l.ID] {
				completed++
			}
		}
		result[i] = ChapterWithProgress{
			Chapter:        ch,
			LessonCount:    len(lessons),
			CompletedCount: completed,
		}
	}

	// Overall course stats
	totalLessons := 0
	totalCompleted := len(progress.CompletedLessonIDs)
	for _, ch := range result {
		totalLessons += ch.LessonCount
	}

	utils.Success(c, http.StatusOK, gin.H{
		"chapters":       result,
		"totalLessons":   totalLessons,
		"completedCount": totalCompleted,
	}, "Success")
}

// GetChapterLessons — GET /chapters/:id/lessons
// Returns lessons for a chapter with the user's completion status.
func GetChapterLessons(c *gin.Context) {
	chapterID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid chapter ID")
		return
	}
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// First get the chapter to find courseId
	var chapter models.Chapter
	if err := config.GetCollection("chapters").FindOne(ctx, bson.M{"_id": chapterID}).Decode(&chapter); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Chapter not found")
		return
	}

	cursor, err := config.GetCollection("lessons").Find(ctx,
		bson.M{"chapterId": chapterID},
		options.Find().SetSort(bson.M{"order": 1}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch lessons")
		return
	}
	defer cursor.Close(ctx)

	var lessons []models.Lesson
	cursor.All(ctx, &lessons)
	if lessons == nil {
		lessons = []models.Lesson{}
	}

	// Fetch user's completed lesson IDs for this course
	var progress models.UserCourseProgress
	completedIDs := []string{}
	if chapter.CourseID != nil {
		if err := config.GetCollection("usercourseprogress").FindOne(ctx,
			bson.M{"userId": userID, "courseId": *chapter.CourseID}).Decode(&progress); err == nil {
			for _, id := range progress.CompletedLessonIDs {
				completedIDs = append(completedIDs, id.Hex())
			}
		}
	}

	utils.Success(c, http.StatusOK, gin.H{
		"lessons":      lessons,
		"completedIds": completedIDs,
	}, "Success")
}

// MarkLessonComplete — POST /lessons/:id/complete
func MarkLessonComplete(c *gin.Context) {
	lessonID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid lesson ID")
		return
	}
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch lesson to get its courseId
	var lesson models.Lesson
	if err := config.GetCollection("lessons").FindOne(ctx, bson.M{"_id": lessonID}).Decode(&lesson); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Lesson not found")
		return
	}

	filter := bson.M{"userId": userID, "courseId": lesson.CourseID}
	update := bson.M{
		"$addToSet": bson.M{"completedLessonIds": lessonID},
		"$set":      bson.M{"updatedAt": time.Now()},
		"$setOnInsert": bson.M{
			"_id": primitive.NewObjectID(),
		},
	}
	opts := options.Update().SetUpsert(true)
	config.GetCollection("usercourseprogress").UpdateOne(ctx, filter, update, opts)

	utils.Success(c, http.StatusOK, nil, "Lesson marked complete")
}
