package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"navodaya-api/config"
	"navodaya-api/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ─── ListCourses ──────────────────────────────────────────────────────────────

func TestListCourses_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	r := newRouter("GET", "/courses", ListCourses)
	w := doRequest(r, "GET", "/courses", nil, nil)

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

func TestListCourses_WithData(t *testing.T) {
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

	r := newRouter("GET", "/courses", ListCourses)
	w := doRequest(r, "GET", "/courses", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	courses := data["courses"].([]interface{})
	if len(courses) != 3 {
		t.Errorf("expected 3 courses, got %d", len(courses))
	}
}

func TestListCourses_FilterBySubject(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	ctx := context.Background()
	config.GetCollection("courses").InsertOne(ctx, models.Course{
		ID:      primitive.NewObjectID(),
		Subject: "Maths",
		Title:   "Maths Course",
	})
	config.GetCollection("courses").InsertOne(ctx, models.Course{
		ID:      primitive.NewObjectID(),
		Subject: "Science",
		Title:   "Science Course",
	})

	r := newRouter("GET", "/courses", ListCourses)
	w := doRequest(r, "GET", "/courses?subject=Maths", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	courses := data["courses"].([]interface{})
	if len(courses) != 1 {
		t.Errorf("expected 1 Maths course, got %d", len(courses))
	}
}

// ─── GetCourse ────────────────────────────────────────────────────────────────

func TestGetCourse_InvalidID(t *testing.T) {
	r := newRouter("GET", "/courses/:id", GetCourse)
	w := doRequest(r, "GET", "/courses/invalid-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestGetCourse_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	id := primitive.NewObjectID()
	r := newRouter("GET", "/courses/:id", GetCourse)
	w := doRequest(r, "GET", "/courses/"+id.Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestGetCourse_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "courses")

	courseID := primitive.NewObjectID()
	ctx := context.Background()
	config.GetCollection("courses").InsertOne(ctx, models.Course{
		ID:      courseID,
		Title:   "Test Course",
		Subject: "Science",
	})

	r := newRouter("GET", "/courses/:id", GetCourse)
	w := doRequest(r, "GET", "/courses/"+courseID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	course := data["course"].(map[string]interface{})
	if course["title"] != "Test Course" {
		t.Errorf("title: want Test Course, got %v", course["title"])
	}
}

// ─── GetCourseChapters ────────────────────────────────────────────────────────

func TestGetCourseChapters_InvalidID(t *testing.T) {
	r := newRouter("GET", "/courses/:id/chapters", GetCourseChapters)
	w := doRequest(r, "GET", "/courses/bad-id/chapters", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestGetCourseChapters_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	courseID := primitive.NewObjectID()
	r := newRouter("GET", "/courses/:id/chapters", GetCourseChapters)
	w := doRequest(r, "GET", "/courses/"+courseID.Hex()+"/chapters", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
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

func TestGetCourseChapters_WithChapters(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	courseID := primitive.NewObjectID()
	ctx := context.Background()
	for i := 1; i <= 2; i++ {
		cid := courseID
		config.GetCollection("chapters").InsertOne(ctx, models.Chapter{
			ID:       primitive.NewObjectID(),
			CourseID: &cid,
			Title:    "Chapter",
			Order:    i,
		})
	}

	r := newRouter("GET", "/courses/:id/chapters", GetCourseChapters)
	w := doRequest(r, "GET", "/courses/"+courseID.Hex()+"/chapters", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	chapters := data["chapters"].([]interface{})
	if len(chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(chapters))
	}
}
