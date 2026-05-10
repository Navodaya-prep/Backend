package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/navodayaprime/api/config"
	"github.com/navodayaprime/api/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ─── AdminListCourses ─────────────────────────────────────────────────────────

func TestAdminListCourses_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/courses",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListCourses,
	)
	w := doRequest(r, "GET", "/admin/courses", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	courses, ok := data["courses"].([]interface{})
	if !ok {
		t.Fatal("expected courses array")
	}
	if len(courses) != 0 {
		t.Errorf("expected 0 courses, got %d", len(courses))
	}
}

func TestAdminListCourses_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	ctx := context.Background()
	for i := 1; i <= 3; i++ {
		config.GetCollection("courses").InsertOne(ctx, models.Course{
			ID:        primitive.NewObjectID(),
			Title:     "Course",
			Subject:   "Maths",
			ClassLevel: "6",
			Order:     i,
			CreatedAt: time.Now(),
		})
	}

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/courses",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListCourses,
	)
	w := doRequest(r, "GET", "/admin/courses", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	courses := data["courses"].([]interface{})
	if len(courses) != 3 {
		t.Errorf("expected 3 courses, got %d", len(courses))
	}
}

// ─── AdminCreateCourse ────────────────────────────────────────────────────────

func TestAdminCreateCourse_MissingTitle(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/courses",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateCourse,
	)
	w := doRequest(r, "POST", "/admin/courses", map[string]interface{}{
		"subject":    "Maths",
		"classLevel": "6",
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing title, got %d", w.Code)
	}
}

func TestAdminCreateCourse_MissingSubject(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/courses",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateCourse,
	)
	w := doRequest(r, "POST", "/admin/courses", map[string]interface{}{
		"title":      "Physics Course",
		"classLevel": "7",
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing subject, got %d", w.Code)
	}
}

func TestAdminCreateCourse_EmptyBody(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/courses",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateCourse,
	)
	// nil body → EOF → 400
	w := doRequest(r, "POST", "/admin/courses", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty body, got %d", w.Code)
	}
}

func TestAdminCreateCourse_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/courses",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateCourse,
	)
	w := doRequest(r, "POST", "/admin/courses", map[string]interface{}{
		"title":       "NCERT Mathematics Class 9",
		"subject":     "Maths",
		"classLevel":  "9",
		"description": "Complete maths course",
		"thumbnail":   "https://example.com/thumb.jpg",
		"isPremium":   false,
		"order":       1,
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	course := data["course"].(map[string]interface{})
	if course["title"] != "NCERT Mathematics Class 9" {
		t.Errorf("title: want NCERT Mathematics Class 9, got %v", course["title"])
	}
}

func TestAdminCreateCourse_MinimalRequiredFields(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/courses",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateCourse,
	)
	// Only title and subject are required (classLevel is optional in handler)
	w := doRequest(r, "POST", "/admin/courses", map[string]interface{}{
		"title":   "Minimal Course",
		"subject": "Science",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201 with minimal required fields, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminUpdateCourse ────────────────────────────────────────────────────────

func TestAdminUpdateCourse_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/courses/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateCourse,
	)
	w := doRequest(r, "PUT", "/admin/courses/bad-id", map[string]interface{}{
		"title":   "Updated",
		"subject": "Science",
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestAdminUpdateCourse_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	adminID := primitive.NewObjectID()
	id := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/courses/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateCourse,
	)
	w := doRequest(r, "PUT", "/admin/courses/"+id.Hex(), map[string]interface{}{
		"title":   "Ghost Course",
		"subject": "Maths",
	}, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent course, got %d", w.Code)
	}
}

func TestAdminUpdateCourse_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	ctx := context.Background()
	course := models.Course{
		ID:        primitive.NewObjectID(),
		Title:     "Old Title",
		Subject:   "Maths",
		ClassLevel: "6",
		CreatedAt: time.Now(),
	}
	config.GetCollection("courses").InsertOne(ctx, course)

	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/courses/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateCourse,
	)
	w := doRequest(r, "PUT", "/admin/courses/"+course.ID.Hex(), map[string]interface{}{
		"title":       "New Title",
		"subject":     "Physics",
		"classLevel":  "7",
		"description": "Updated description",
		"isPremium":   true,
		"order":       3,
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminDeleteCourse ────────────────────────────────────────────────────────

func TestAdminDeleteCourse_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/courses/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteCourse,
	)
	w := doRequest(r, "DELETE", "/admin/courses/bad-id", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestAdminDeleteCourse_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/courses/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteCourse,
	)
	w := doRequest(r, "DELETE", "/admin/courses/"+primitive.NewObjectID().Hex(), nil, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent course, got %d", w.Code)
	}
}

func TestAdminDeleteCourse_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	ctx := context.Background()
	course := models.Course{
		ID:        primitive.NewObjectID(),
		Title:     "Delete Me",
		Subject:   "Maths",
		CreatedAt: time.Now(),
	}
	config.GetCollection("courses").InsertOne(ctx, course)

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/courses/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteCourse,
	)
	w := doRequest(r, "DELETE", "/admin/courses/"+course.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminListCourseChapters ──────────────────────────────────────────────────

func TestAdminListCourseChapters_InvalidCourseID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/courses/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListCourseChapters,
	)
	w := doRequest(r, "GET", "/admin/courses/bad-id/chapters", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid course ID, got %d", w.Code)
	}
}

func TestAdminListCourseChapters_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	courseID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/courses/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListCourseChapters,
	)
	w := doRequest(r, "GET", "/admin/courses/"+courseID.Hex()+"/chapters", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	chapters, ok := data["chapters"].([]interface{})
	if !ok {
		t.Fatal("expected chapters array")
	}
	if len(chapters) != 0 {
		t.Errorf("expected 0 chapters, got %d", len(chapters))
	}
}

func TestAdminListCourseChapters_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	courseID := primitive.NewObjectID()
	ctx := context.Background()
	for i := 1; i <= 3; i++ {
		cid := courseID
		config.GetCollection("chapters").InsertOne(ctx, models.Chapter{
			ID:        primitive.NewObjectID(),
			CourseID:  &cid,
			Title:     "Chapter",
			Order:     i,
			CreatedAt: time.Now(),
		})
	}

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/courses/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListCourseChapters,
	)
	w := doRequest(r, "GET", "/admin/courses/"+courseID.Hex()+"/chapters", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	chapters := data["chapters"].([]interface{})
	if len(chapters) != 3 {
		t.Errorf("expected 3 chapters, got %d", len(chapters))
	}
}

func TestAdminListCourseChapters_OnlyReturnsCourseChapters(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	courseID := primitive.NewObjectID()
	otherCourseID := primitive.NewObjectID()
	ctx := context.Background()

	// Insert chapter for the target course
	cid := courseID
	config.GetCollection("chapters").InsertOne(ctx, models.Chapter{
		ID: primitive.NewObjectID(), CourseID: &cid, Title: "Target Chapter", CreatedAt: time.Now(),
	})
	// Insert chapter for a different course (should NOT appear)
	oid := otherCourseID
	config.GetCollection("chapters").InsertOne(ctx, models.Chapter{
		ID: primitive.NewObjectID(), CourseID: &oid, Title: "Other Chapter", CreatedAt: time.Now(),
	})

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/courses/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListCourseChapters,
	)
	w := doRequest(r, "GET", "/admin/courses/"+courseID.Hex()+"/chapters", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	chapters := data["chapters"].([]interface{})
	if len(chapters) != 1 {
		t.Errorf("expected 1 chapter for the course, got %d", len(chapters))
	}
}

// ─── AdminCreateCourseChapter ─────────────────────────────────────────────────

func TestAdminCreateCourseChapter_InvalidCourseID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/courses/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateCourseChapter,
	)
	w := doRequest(r, "POST", "/admin/courses/bad-id/chapters", map[string]interface{}{
		"title": "Chapter One",
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid course ID, got %d", w.Code)
	}
}

func TestAdminCreateCourseChapter_MissingTitle(t *testing.T) {
	courseID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/courses/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateCourseChapter,
	)
	w := doRequest(r, "POST", "/admin/courses/"+courseID.Hex()+"/chapters",
		map[string]interface{}{
			"description": "Chapter without a title",
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing title, got %d", w.Code)
	}
}

func TestAdminCreateCourseChapter_EmptyBody(t *testing.T) {
	courseID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/courses/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateCourseChapter,
	)
	w := doRequest(r, "POST", "/admin/courses/"+courseID.Hex()+"/chapters", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty body, got %d", w.Code)
	}
}

func TestAdminCreateCourseChapter_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")
	dropCollection(t, "courses")

	ctx := context.Background()
	course := models.Course{
		ID:        primitive.NewObjectID(),
		Title:     "Test Course",
		Subject:   "Maths",
		CreatedAt: time.Now(),
	}
	config.GetCollection("courses").InsertOne(ctx, course)

	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/courses/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateCourseChapter,
	)
	w := doRequest(r, "POST", "/admin/courses/"+course.ID.Hex()+"/chapters",
		map[string]interface{}{
			"title":       "Introduction Chapter",
			"description": "First chapter of the course",
			"order":       1,
			"isPremium":   false,
		}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	chapter := data["chapter"].(map[string]interface{})
	if chapter["title"] != "Introduction Chapter" {
		t.Errorf("title: want Introduction Chapter, got %v", chapter["title"])
	}
}

// ─── AdminListLessons ─────────────────────────────────────────────────────────

func TestAdminListLessons_InvalidChapterID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListLessons,
	)
	w := doRequest(r, "GET", "/admin/chapters/bad-id/lessons", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid chapter ID, got %d", w.Code)
	}
}

func TestAdminListLessons_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")

	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListLessons,
	)
	w := doRequest(r, "GET", "/admin/chapters/"+chapterID.Hex()+"/lessons", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	lessons, ok := data["lessons"].([]interface{})
	if !ok {
		t.Fatal("expected lessons array")
	}
	if len(lessons) != 0 {
		t.Errorf("expected 0 lessons, got %d", len(lessons))
	}
}

func TestAdminListLessons_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")

	chapterID := primitive.NewObjectID()
	courseID := primitive.NewObjectID()
	ctx := context.Background()
	for i := 1; i <= 3; i++ {
		config.GetCollection("lessons").InsertOne(ctx, models.Lesson{
			ID:             primitive.NewObjectID(),
			ChapterID:      chapterID,
			CourseID:       courseID,
			Title:          "Lesson",
			Type:           "video",
			YouTubeVideoID: "abc123",
			Order:          i,
			CreatedAt:      time.Now(),
		})
	}

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListLessons,
	)
	w := doRequest(r, "GET", "/admin/chapters/"+chapterID.Hex()+"/lessons", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	lessons := data["lessons"].([]interface{})
	if len(lessons) != 3 {
		t.Errorf("expected 3 lessons, got %d", len(lessons))
	}
}

// ─── AdminCreateLesson ────────────────────────────────────────────────────────

func TestAdminCreateLesson_InvalidChapterID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateLesson,
	)
	w := doRequest(r, "POST", "/admin/chapters/bad-id/lessons", map[string]interface{}{
		"title":    "Lesson 1",
		"type":     "video",
		"courseId": primitive.NewObjectID().Hex(),
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid chapter ID, got %d", w.Code)
	}
}

func TestAdminCreateLesson_MissingTitle(t *testing.T) {
	chapterID := primitive.NewObjectID()
	courseID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateLesson,
	)
	w := doRequest(r, "POST", "/admin/chapters/"+chapterID.Hex()+"/lessons",
		map[string]interface{}{
			"type":     "video",
			"courseId": courseID.Hex(),
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing title, got %d", w.Code)
	}
}

func TestAdminCreateLesson_MissingType(t *testing.T) {
	chapterID := primitive.NewObjectID()
	courseID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateLesson,
	)
	w := doRequest(r, "POST", "/admin/chapters/"+chapterID.Hex()+"/lessons",
		map[string]interface{}{
			"title":    "Lesson 1",
			"courseId": courseID.Hex(),
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing type, got %d", w.Code)
	}
}

func TestAdminCreateLesson_MissingCourseID(t *testing.T) {
	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateLesson,
	)
	w := doRequest(r, "POST", "/admin/chapters/"+chapterID.Hex()+"/lessons",
		map[string]interface{}{
			"title": "Lesson 1",
			"type":  "video",
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing courseId, got %d", w.Code)
	}
}

func TestAdminCreateLesson_InvalidType(t *testing.T) {
	chapterID := primitive.NewObjectID()
	courseID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateLesson,
	)
	w := doRequest(r, "POST", "/admin/chapters/"+chapterID.Hex()+"/lessons",
		map[string]interface{}{
			"title":    "Lesson 1",
			"type":     "invalid-type",
			"courseId": courseID.Hex(),
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid type, got %d", w.Code)
	}
}

func TestAdminCreateLesson_InvalidCourseIDFormat(t *testing.T) {
	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateLesson,
	)
	w := doRequest(r, "POST", "/admin/chapters/"+chapterID.Hex()+"/lessons",
		map[string]interface{}{
			"title":    "Lesson 1",
			"type":     "video",
			"courseId": "not-a-valid-id",
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid courseId format, got %d", w.Code)
	}
}

func TestAdminCreateLesson_EmptyBody(t *testing.T) {
	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateLesson,
	)
	w := doRequest(r, "POST", "/admin/chapters/"+chapterID.Hex()+"/lessons", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty body, got %d", w.Code)
	}
}

func TestAdminCreateLesson_SuccessVideoType(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")
	dropCollection(t, "courses")

	ctx := context.Background()
	courseID := primitive.NewObjectID()
	chapterID := primitive.NewObjectID()
	config.GetCollection("courses").InsertOne(ctx, models.Course{
		ID: courseID, Title: "Test", Subject: "Maths", CreatedAt: time.Now(),
	})

	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateLesson,
	)
	w := doRequest(r, "POST", "/admin/chapters/"+chapterID.Hex()+"/lessons",
		map[string]interface{}{
			"title":          "Introduction to Algebra",
			"type":           "video",
			"youtubeVideoId": "dQw4w9WgXcQ",
			"description":    "First video lesson",
			"durationMins":   20,
			"order":          1,
			"isPremium":      false,
			"courseId":       courseID.Hex(),
		}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	lesson := data["lesson"].(map[string]interface{})
	if lesson["title"] != "Introduction to Algebra" {
		t.Errorf("title: want Introduction to Algebra, got %v", lesson["title"])
	}
	if lesson["type"] != "video" {
		t.Errorf("type: want video, got %v", lesson["type"])
	}
}

func TestAdminCreateLesson_SuccessNoteType(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")

	courseID := primitive.NewObjectID()
	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/chapters/:id/lessons",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateLesson,
	)
	w := doRequest(r, "POST", "/admin/chapters/"+chapterID.Hex()+"/lessons",
		map[string]interface{}{
			"title":       "Chapter Notes",
			"type":        "note",
			"noteContent": "# Chapter 1\nThis is the content of the note.",
			"order":       2,
			"courseId":    courseID.Hex(),
		}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201 for note type, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminUpdateLesson ────────────────────────────────────────────────────────

func TestAdminUpdateLesson_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/lessons/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateLesson,
	)
	w := doRequest(r, "PUT", "/admin/lessons/bad-id", map[string]interface{}{
		"title": "Updated Lesson",
		"type":  "video",
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid lesson ID, got %d", w.Code)
	}
}

func TestAdminUpdateLesson_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")

	adminID := primitive.NewObjectID()
	id := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/lessons/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateLesson,
	)
	w := doRequest(r, "PUT", "/admin/lessons/"+id.Hex(), map[string]interface{}{
		"title": "Ghost Lesson",
		"type":  "video",
	}, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent lesson, got %d", w.Code)
	}
}

func TestAdminUpdateLesson_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")

	chapterID := primitive.NewObjectID()
	courseID := primitive.NewObjectID()
	ctx := context.Background()
	lesson := models.Lesson{
		ID:             primitive.NewObjectID(),
		ChapterID:      chapterID,
		CourseID:       courseID,
		Title:          "Old Lesson Title",
		Type:           "video",
		YouTubeVideoID: "oldVideoID",
		Order:          1,
		CreatedAt:      time.Now(),
	}
	config.GetCollection("lessons").InsertOne(ctx, lesson)

	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/lessons/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateLesson,
	)
	w := doRequest(r, "PUT", "/admin/lessons/"+lesson.ID.Hex(), map[string]interface{}{
		"title":          "New Lesson Title",
		"type":           "video",
		"youtubeVideoId": "newVideoID",
		"description":    "Updated description",
		"durationMins":   30,
		"order":          2,
		"isPremium":      true,
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminDeleteLesson ────────────────────────────────────────────────────────

func TestAdminDeleteLesson_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/lessons/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteLesson,
	)
	w := doRequest(r, "DELETE", "/admin/lessons/bad-id", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid lesson ID, got %d", w.Code)
	}
}

func TestAdminDeleteLesson_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/lessons/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteLesson,
	)
	w := doRequest(r, "DELETE", "/admin/lessons/"+primitive.NewObjectID().Hex(), nil, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent lesson, got %d", w.Code)
	}
}

func TestAdminDeleteLesson_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")

	chapterID := primitive.NewObjectID()
	courseID := primitive.NewObjectID()
	ctx := context.Background()
	lesson := models.Lesson{
		ID:        primitive.NewObjectID(),
		ChapterID: chapterID,
		CourseID:  courseID,
		Title:     "Delete This Lesson",
		Type:      "video",
		CreatedAt: time.Now(),
	}
	config.GetCollection("lessons").InsertOne(ctx, lesson)

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/lessons/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteLesson,
	)
	w := doRequest(r, "DELETE", "/admin/lessons/"+lesson.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── GetCourseChaptersWithProgress (Student) ─────────────────────────────────

func TestGetCourseChaptersWithProgress_InvalidCourseID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("GET", "/courses/:id/chapters/progress",
		setUserID(userID.Hex(), "9876543210"),
		GetCourseChaptersWithProgress,
	)
	w := doRequest(r, "GET", "/courses/bad-id/chapters/progress", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid course ID, got %d", w.Code)
	}
}

func TestGetCourseChaptersWithProgress_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")
	dropCollection(t, "lessons")
	dropCollection(t, "usercourseprogress")

	courseID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	r := newRouter("GET", "/courses/:id/chapters/progress",
		setUserID(userID.Hex(), "9876543210"),
		GetCourseChaptersWithProgress,
	)
	w := doRequest(r, "GET", "/courses/"+courseID.Hex()+"/chapters/progress", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	chapters, ok := data["chapters"].([]interface{})
	if !ok {
		t.Fatal("expected chapters array")
	}
	if len(chapters) != 0 {
		t.Errorf("expected 0 chapters, got %d", len(chapters))
	}
}

func TestGetCourseChaptersWithProgress_WithChapters(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")
	dropCollection(t, "lessons")
	dropCollection(t, "usercourseprogress")

	courseID := primitive.NewObjectID()
	ctx := context.Background()
	for i := 1; i <= 2; i++ {
		cid := courseID
		config.GetCollection("chapters").InsertOne(ctx, models.Chapter{
			ID:        primitive.NewObjectID(),
			CourseID:  &cid,
			Title:     "Chapter",
			Order:     i,
			CreatedAt: time.Now(),
		})
	}

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/courses/:id/chapters/progress",
		setUserID(userID.Hex(), "9876543210"),
		GetCourseChaptersWithProgress,
	)
	w := doRequest(r, "GET", "/courses/"+courseID.Hex()+"/chapters/progress", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	chapters := data["chapters"].([]interface{})
	if len(chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(chapters))
	}
}

func TestGetCourseChaptersWithProgress_TracksCompletedLessons(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")
	dropCollection(t, "lessons")
	dropCollection(t, "usercourseprogress")

	courseID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	ctx := context.Background()

	cid := courseID
	chapter := models.Chapter{
		ID:        primitive.NewObjectID(),
		CourseID:  &cid,
		Title:     "Chapter with Lessons",
		Order:     1,
		CreatedAt: time.Now(),
	}
	config.GetCollection("chapters").InsertOne(ctx, chapter)

	// Insert 3 lessons for the chapter
	lesson1 := models.Lesson{
		ID: primitive.NewObjectID(), ChapterID: chapter.ID, CourseID: courseID,
		Title: "L1", Type: "video", Order: 1, CreatedAt: time.Now(),
	}
	lesson2 := models.Lesson{
		ID: primitive.NewObjectID(), ChapterID: chapter.ID, CourseID: courseID,
		Title: "L2", Type: "video", Order: 2, CreatedAt: time.Now(),
	}
	lesson3 := models.Lesson{
		ID: primitive.NewObjectID(), ChapterID: chapter.ID, CourseID: courseID,
		Title: "L3", Type: "note", Order: 3, CreatedAt: time.Now(),
	}
	config.GetCollection("lessons").InsertOne(ctx, lesson1)
	config.GetCollection("lessons").InsertOne(ctx, lesson2)
	config.GetCollection("lessons").InsertOne(ctx, lesson3)

	// Mark lessons 1 and 2 as complete for this user
	config.GetCollection("usercourseprogress").InsertOne(ctx, models.UserCourseProgress{
		ID:                 primitive.NewObjectID(),
		UserID:             userID,
		CourseID:           courseID,
		CompletedLessonIDs: []primitive.ObjectID{lesson1.ID, lesson2.ID},
		UpdatedAt:          time.Now(),
	})

	r := newRouter("GET", "/courses/:id/chapters/progress",
		setUserID(userID.Hex(), "9876543210"),
		GetCourseChaptersWithProgress,
	)
	w := doRequest(r, "GET", "/courses/"+courseID.Hex()+"/chapters/progress", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	chapters := data["chapters"].([]interface{})
	if len(chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(chapters))
	}
	ch := chapters[0].(map[string]interface{})
	if ch["lessonCount"].(float64) != 3 {
		t.Errorf("lessonCount: want 3, got %v", ch["lessonCount"])
	}
	if ch["completedCount"].(float64) != 2 {
		t.Errorf("completedCount: want 2, got %v", ch["completedCount"])
	}
	if data["totalLessons"].(float64) != 3 {
		t.Errorf("totalLessons: want 3, got %v", data["totalLessons"])
	}
	if data["completedCount"].(float64) != 2 {
		t.Errorf("overall completedCount: want 2, got %v", data["completedCount"])
	}
}

// ─── GetChapterLessons (Student) ──────────────────────────────────────────────

func TestGetChapterLessons_InvalidChapterID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("GET", "/chapters/:id/lessons",
		setUserID(userID.Hex(), "9876543210"),
		GetChapterLessons,
	)
	w := doRequest(r, "GET", "/chapters/bad-id/lessons", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid chapter ID, got %d", w.Code)
	}
}

func TestGetChapterLessons_ChapterNotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/chapters/:id/lessons",
		setUserID(userID.Hex(), "9876543210"),
		GetChapterLessons,
	)
	w := doRequest(r, "GET", "/chapters/"+primitive.NewObjectID().Hex()+"/lessons", nil, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent chapter, got %d", w.Code)
	}
}

func TestGetChapterLessons_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")
	dropCollection(t, "lessons")
	dropCollection(t, "usercourseprogress")

	courseID := primitive.NewObjectID()
	ctx := context.Background()
	cid := courseID
	chapter := models.Chapter{
		ID:        primitive.NewObjectID(),
		CourseID:  &cid,
		Title:     "Empty Chapter",
		Order:     1,
		CreatedAt: time.Now(),
	}
	config.GetCollection("chapters").InsertOne(ctx, chapter)

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/chapters/:id/lessons",
		setUserID(userID.Hex(), "9876543210"),
		GetChapterLessons,
	)
	w := doRequest(r, "GET", "/chapters/"+chapter.ID.Hex()+"/lessons", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	lessons, ok := data["lessons"].([]interface{})
	if !ok {
		t.Fatal("expected lessons array")
	}
	if len(lessons) != 0 {
		t.Errorf("expected 0 lessons, got %d", len(lessons))
	}
}

func TestGetChapterLessons_WithLessons(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")
	dropCollection(t, "lessons")
	dropCollection(t, "usercourseprogress")

	courseID := primitive.NewObjectID()
	ctx := context.Background()
	cid := courseID
	chapter := models.Chapter{
		ID:        primitive.NewObjectID(),
		CourseID:  &cid,
		Title:     "Chapter With Lessons",
		Order:     1,
		CreatedAt: time.Now(),
	}
	config.GetCollection("chapters").InsertOne(ctx, chapter)

	for i := 1; i <= 4; i++ {
		config.GetCollection("lessons").InsertOne(ctx, models.Lesson{
			ID:             primitive.NewObjectID(),
			ChapterID:      chapter.ID,
			CourseID:       courseID,
			Title:          "Lesson",
			Type:           "video",
			YouTubeVideoID: "vid123",
			Order:          i,
			CreatedAt:      time.Now(),
		})
	}

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/chapters/:id/lessons",
		setUserID(userID.Hex(), "9876543210"),
		GetChapterLessons,
	)
	w := doRequest(r, "GET", "/chapters/"+chapter.ID.Hex()+"/lessons", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	lessons := data["lessons"].([]interface{})
	if len(lessons) != 4 {
		t.Errorf("expected 4 lessons, got %d", len(lessons))
	}
}

func TestGetChapterLessons_IncludesCompletedIDs(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")
	dropCollection(t, "lessons")
	dropCollection(t, "usercourseprogress")

	courseID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	ctx := context.Background()

	cid := courseID
	chapter := models.Chapter{
		ID:        primitive.NewObjectID(),
		CourseID:  &cid,
		Title:     "Progress Chapter",
		Order:     1,
		CreatedAt: time.Now(),
	}
	config.GetCollection("chapters").InsertOne(ctx, chapter)

	lesson := models.Lesson{
		ID:        primitive.NewObjectID(),
		ChapterID: chapter.ID,
		CourseID:  courseID,
		Title:     "Completed Lesson",
		Type:      "video",
		Order:     1,
		CreatedAt: time.Now(),
	}
	config.GetCollection("lessons").InsertOne(ctx, lesson)

	// Record the lesson as completed for this user
	config.GetCollection("usercourseprogress").InsertOne(ctx, models.UserCourseProgress{
		ID:                 primitive.NewObjectID(),
		UserID:             userID,
		CourseID:           courseID,
		CompletedLessonIDs: []primitive.ObjectID{lesson.ID},
		UpdatedAt:          time.Now(),
	})

	r := newRouter("GET", "/chapters/:id/lessons",
		setUserID(userID.Hex(), "9876543210"),
		GetChapterLessons,
	)
	w := doRequest(r, "GET", "/chapters/"+chapter.ID.Hex()+"/lessons", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	completedIDs, ok := data["completedIds"].([]interface{})
	if !ok {
		t.Fatal("expected completedIds array")
	}
	if len(completedIDs) != 1 {
		t.Errorf("expected 1 completed ID, got %d", len(completedIDs))
	}
	if completedIDs[0] != lesson.ID.Hex() {
		t.Errorf("completedId: want %s, got %v", lesson.ID.Hex(), completedIDs[0])
	}
}

// ─── MarkLessonComplete (Student) ────────────────────────────────────────────

func TestMarkLessonComplete_InvalidLessonID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/lessons/:id/complete",
		setUserID(userID.Hex(), "9876543210"),
		MarkLessonComplete,
	)
	w := doRequest(r, "POST", "/lessons/bad-id/complete", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid lesson ID, got %d", w.Code)
	}
}

func TestMarkLessonComplete_LessonNotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/lessons/:id/complete",
		setUserID(userID.Hex(), "9876543210"),
		MarkLessonComplete,
	)
	w := doRequest(r, "POST", "/lessons/"+primitive.NewObjectID().Hex()+"/complete", nil, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent lesson, got %d", w.Code)
	}
}

func TestMarkLessonComplete_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")
	dropCollection(t, "usercourseprogress")

	chapterID := primitive.NewObjectID()
	courseID := primitive.NewObjectID()
	ctx := context.Background()
	lesson := models.Lesson{
		ID:        primitive.NewObjectID(),
		ChapterID: chapterID,
		CourseID:  courseID,
		Title:     "Lesson to Complete",
		Type:      "video",
		Order:     1,
		CreatedAt: time.Now(),
	}
	config.GetCollection("lessons").InsertOne(ctx, lesson)

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/lessons/:id/complete",
		setUserID(userID.Hex(), "9876543210"),
		MarkLessonComplete,
	)
	w := doRequest(r, "POST", "/lessons/"+lesson.ID.Hex()+"/complete", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

func TestMarkLessonComplete_Idempotent(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")
	dropCollection(t, "usercourseprogress")

	chapterID := primitive.NewObjectID()
	courseID := primitive.NewObjectID()
	ctx := context.Background()
	lesson := models.Lesson{
		ID:        primitive.NewObjectID(),
		ChapterID: chapterID,
		CourseID:  courseID,
		Title:     "Idempotent Lesson",
		Type:      "video",
		Order:     1,
		CreatedAt: time.Now(),
	}
	config.GetCollection("lessons").InsertOne(ctx, lesson)

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/lessons/:id/complete",
		setUserID(userID.Hex(), "9876543210"),
		MarkLessonComplete,
	)

	// First completion
	w1 := doRequest(r, "POST", "/lessons/"+lesson.ID.Hex()+"/complete", nil, nil)
	if w1.Code != http.StatusOK {
		t.Errorf("first call: want 200, got %d", w1.Code)
	}

	// Second completion — upsert with $addToSet is idempotent
	w2 := doRequest(r, "POST", "/lessons/"+lesson.ID.Hex()+"/complete", nil, nil)
	if w2.Code != http.StatusOK {
		t.Errorf("second call: want 200, got %d", w2.Code)
	}

	// Verify only one entry in completedLessonIds
	var progress models.UserCourseProgress
	err := config.GetCollection("usercourseprogress").FindOne(ctx, map[string]interface{}{
		"userId":   userID,
		"courseId": courseID,
	}).Decode(&progress)
	if err != nil {
		t.Errorf("expected usercourseprogress record: %v", err)
	}
	if len(progress.CompletedLessonIDs) != 1 {
		t.Errorf("expected 1 completed lesson ID after double completion, got %d", len(progress.CompletedLessonIDs))
	}
}

func TestMarkLessonComplete_CreatesProgressRecord(t *testing.T) {
	requireDB(t)
	dropCollection(t, "lessons")
	dropCollection(t, "usercourseprogress")

	chapterID := primitive.NewObjectID()
	courseID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	ctx := context.Background()

	lesson := models.Lesson{
		ID:        primitive.NewObjectID(),
		ChapterID: chapterID,
		CourseID:  courseID,
		Title:     "Progress Lesson",
		Type:      "video",
		Order:     1,
		CreatedAt: time.Now(),
	}
	config.GetCollection("lessons").InsertOne(ctx, lesson)

	r := newRouter("POST", "/lessons/:id/complete",
		setUserID(userID.Hex(), "9876543210"),
		MarkLessonComplete,
	)
	w := doRequest(r, "POST", "/lessons/"+lesson.ID.Hex()+"/complete", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify the progress record was created/updated
	var progress models.UserCourseProgress
	err := config.GetCollection("usercourseprogress").FindOne(ctx, map[string]interface{}{
		"userId":   userID,
		"courseId": courseID,
	}).Decode(&progress)
	if err != nil {
		t.Fatalf("expected usercourseprogress record to be created: %v", err)
	}
	if len(progress.CompletedLessonIDs) != 1 {
		t.Errorf("expected 1 completed lesson ID, got %d", len(progress.CompletedLessonIDs))
	}
	if progress.CompletedLessonIDs[0] != lesson.ID {
		t.Errorf("completedLessonId: want %s, got %s", lesson.ID.Hex(), progress.CompletedLessonIDs[0].Hex())
	}
}
