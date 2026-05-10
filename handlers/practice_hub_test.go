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

// ─── AdminListSubjects ────────────────────────────────────────────────────────

func TestAdminListSubjects_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "subjects")

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/practice/subjects",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListSubjects,
	)
	w := doRequest(r, "GET", "/admin/practice/subjects", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	subjects, ok := data["subjects"].([]interface{})
	if !ok {
		t.Fatal("expected subjects array")
	}
	if len(subjects) != 0 {
		t.Errorf("expected 0 subjects, got %d", len(subjects))
	}
}

func TestAdminListSubjects_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "subjects")

	ctx := context.Background()
	for i := 1; i <= 3; i++ {
		config.GetCollection("subjects").InsertOne(ctx, models.Subject{
			ID:        primitive.NewObjectID(),
			Name:      "Subject",
			Order:     i,
			CreatedAt: time.Now(),
		})
	}

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/practice/subjects",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListSubjects,
	)
	w := doRequest(r, "GET", "/admin/practice/subjects", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	subjects := data["subjects"].([]interface{})
	if len(subjects) != 3 {
		t.Errorf("expected 3 subjects, got %d", len(subjects))
	}
}

// ─── AdminCreateSubject ───────────────────────────────────────────────────────

func TestAdminCreateSubject_MissingName(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/subjects",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateSubject,
	)
	w := doRequest(r, "POST", "/admin/practice/subjects", map[string]interface{}{
		"description": "A subject without a name",
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing name, got %d", w.Code)
	}
}

func TestAdminCreateSubject_EmptyBody(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/subjects",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateSubject,
	)
	// nil body → empty → ShouldBindJSON EOF → 400
	w := doRequest(r, "POST", "/admin/practice/subjects", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty body, got %d", w.Code)
	}
}

func TestAdminCreateSubject_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "subjects")

	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/subjects",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateSubject,
	)
	w := doRequest(r, "POST", "/admin/practice/subjects", map[string]interface{}{
		"name":        "Mathematics",
		"description": "Numbers and equations",
		"icon":        "math-icon",
		"color":       "#FF5733",
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
	subject := data["subject"].(map[string]interface{})
	if subject["name"] != "Mathematics" {
		t.Errorf("name: want Mathematics, got %v", subject["name"])
	}
}

func TestAdminCreateSubject_MinimalBody(t *testing.T) {
	requireDB(t)
	dropCollection(t, "subjects")

	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/subjects",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateSubject,
	)
	// Only required field
	w := doRequest(r, "POST", "/admin/practice/subjects", map[string]interface{}{
		"name": "Science",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201 with only required field, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminUpdateSubject ───────────────────────────────────────────────────────

func TestAdminUpdateSubject_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/practice/subjects/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateSubject,
	)
	w := doRequest(r, "PUT", "/admin/practice/subjects/not-an-id", map[string]interface{}{
		"name": "Updated",
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestAdminUpdateSubject_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "subjects")

	adminID := primitive.NewObjectID()
	id := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/practice/subjects/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateSubject,
	)
	w := doRequest(r, "PUT", "/admin/practice/subjects/"+id.Hex(), map[string]interface{}{
		"name": "Ghost Subject",
	}, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent subject, got %d", w.Code)
	}
}

func TestAdminUpdateSubject_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "subjects")

	ctx := context.Background()
	subject := models.Subject{
		ID:        primitive.NewObjectID(),
		Name:      "Old Name",
		CreatedAt: time.Now(),
	}
	config.GetCollection("subjects").InsertOne(ctx, subject)

	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/practice/subjects/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateSubject,
	)
	w := doRequest(r, "PUT", "/admin/practice/subjects/"+subject.ID.Hex(), map[string]interface{}{
		"name":  "New Name",
		"icon":  "new-icon",
		"color": "#000000",
		"order": 2,
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminDeleteSubject ───────────────────────────────────────────────────────

func TestAdminDeleteSubject_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/practice/subjects/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteSubject,
	)
	w := doRequest(r, "DELETE", "/admin/practice/subjects/bad-id", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestAdminDeleteSubject_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "subjects")

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/practice/subjects/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteSubject,
	)
	w := doRequest(r, "DELETE", "/admin/practice/subjects/"+primitive.NewObjectID().Hex(), nil, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent subject, got %d", w.Code)
	}
}

func TestAdminDeleteSubject_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "subjects")

	ctx := context.Background()
	subject := models.Subject{
		ID:        primitive.NewObjectID(),
		Name:      "Delete Me",
		CreatedAt: time.Now(),
	}
	config.GetCollection("subjects").InsertOne(ctx, subject)

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/practice/subjects/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteSubject,
	)
	w := doRequest(r, "DELETE", "/admin/practice/subjects/"+subject.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminListChapters ────────────────────────────────────────────────────────

func TestAdminListChapters_InvalidSubjectID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/practice/subjects/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListChapters,
	)
	w := doRequest(r, "GET", "/admin/practice/subjects/bad-id/chapters", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid subjectID, got %d", w.Code)
	}
}

func TestAdminListChapters_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	subjectID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/practice/subjects/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListChapters,
	)
	w := doRequest(r, "GET", "/admin/practice/subjects/"+subjectID.Hex()+"/chapters", nil, nil)

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

func TestAdminListChapters_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	subjectID := primitive.NewObjectID()
	ctx := context.Background()
	for i := 1; i <= 2; i++ {
		sid := subjectID
		config.GetCollection("chapters").InsertOne(ctx, models.Chapter{
			ID:        primitive.NewObjectID(),
			SubjectID: &sid,
			Title:     "Chapter",
			Order:     i,
			CreatedAt: time.Now(),
		})
	}

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/practice/subjects/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListChapters,
	)
	w := doRequest(r, "GET", "/admin/practice/subjects/"+subjectID.Hex()+"/chapters", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	chapters := data["chapters"].([]interface{})
	if len(chapters) != 2 {
		t.Errorf("expected 2 chapters, got %d", len(chapters))
	}
}

// ─── AdminCreateChapter ───────────────────────────────────────────────────────

func TestAdminCreateChapter_InvalidSubjectID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/subjects/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateChapter,
	)
	w := doRequest(r, "POST", "/admin/practice/subjects/bad-id/chapters", map[string]interface{}{
		"title": "A Chapter",
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid subjectID, got %d", w.Code)
	}
}

func TestAdminCreateChapter_MissingTitle(t *testing.T) {
	subjectID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/subjects/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateChapter,
	)
	w := doRequest(r, "POST", "/admin/practice/subjects/"+subjectID.Hex()+"/chapters",
		map[string]interface{}{
			"description": "missing title",
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing title, got %d", w.Code)
	}
}

func TestAdminCreateChapter_EmptyBody(t *testing.T) {
	subjectID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/subjects/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateChapter,
	)
	w := doRequest(r, "POST", "/admin/practice/subjects/"+subjectID.Hex()+"/chapters", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty body, got %d", w.Code)
	}
}

func TestAdminCreateChapter_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	subjectID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/subjects/:id/chapters",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateChapter,
	)
	w := doRequest(r, "POST", "/admin/practice/subjects/"+subjectID.Hex()+"/chapters",
		map[string]interface{}{
			"title":       "Algebra Basics",
			"description": "Introduction to algebra",
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
	if chapter["title"] != "Algebra Basics" {
		t.Errorf("title: want Algebra Basics, got %v", chapter["title"])
	}
}

// ─── AdminUpdateChapter ───────────────────────────────────────────────────────

func TestAdminUpdateChapter_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/practice/chapters/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateChapter,
	)
	w := doRequest(r, "PUT", "/admin/practice/chapters/bad-id", map[string]interface{}{
		"title": "Updated",
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid chapter ID, got %d", w.Code)
	}
}

func TestAdminUpdateChapter_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	adminID := primitive.NewObjectID()
	id := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/practice/chapters/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateChapter,
	)
	w := doRequest(r, "PUT", "/admin/practice/chapters/"+id.Hex(), map[string]interface{}{
		"title": "Ghost Chapter",
	}, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent chapter, got %d", w.Code)
	}
}

func TestAdminUpdateChapter_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	subjectID := primitive.NewObjectID()
	ctx := context.Background()
	chapter := models.Chapter{
		ID:        primitive.NewObjectID(),
		SubjectID: &subjectID,
		Title:     "Old Chapter Title",
		Order:     1,
		CreatedAt: time.Now(),
	}
	config.GetCollection("chapters").InsertOne(ctx, chapter)

	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/practice/chapters/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateChapter,
	)
	w := doRequest(r, "PUT", "/admin/practice/chapters/"+chapter.ID.Hex(), map[string]interface{}{
		"title":     "New Chapter Title",
		"order":     2,
		"isPremium": true,
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminDeleteChapter ───────────────────────────────────────────────────────

func TestAdminDeleteChapter_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/practice/chapters/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteChapter,
	)
	w := doRequest(r, "DELETE", "/admin/practice/chapters/bad-id", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid chapter ID, got %d", w.Code)
	}
}

func TestAdminDeleteChapter_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/practice/chapters/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteChapter,
	)
	w := doRequest(r, "DELETE", "/admin/practice/chapters/"+primitive.NewObjectID().Hex(), nil, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent chapter, got %d", w.Code)
	}
}

func TestAdminDeleteChapter_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")

	subjectID := primitive.NewObjectID()
	ctx := context.Background()
	chapter := models.Chapter{
		ID:        primitive.NewObjectID(),
		SubjectID: &subjectID,
		Title:     "Delete This Chapter",
		CreatedAt: time.Now(),
	}
	config.GetCollection("chapters").InsertOne(ctx, chapter)

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/practice/chapters/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteChapter,
	)
	w := doRequest(r, "DELETE", "/admin/practice/chapters/"+chapter.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminListChapterQuestions ────────────────────────────────────────────────

func TestAdminListChapterQuestions_InvalidChapterID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListChapterQuestions,
	)
	w := doRequest(r, "GET", "/admin/practice/chapters/bad-id/questions", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid chapter ID, got %d", w.Code)
	}
}

func TestAdminListChapterQuestions_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListChapterQuestions,
	)
	w := doRequest(r, "GET", "/admin/practice/chapters/"+chapterID.Hex()+"/questions", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	questions, ok := data["questions"].([]interface{})
	if !ok {
		t.Fatal("expected questions array")
	}
	if len(questions) != 0 {
		t.Errorf("expected 0 questions, got %d", len(questions))
	}
}

func TestAdminListChapterQuestions_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	chapterID := primitive.NewObjectID()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		cid := chapterID
		config.GetCollection("questions").InsertOne(ctx, models.Question{
			ID:        primitive.NewObjectID(),
			ChapterID: &cid,
			Text:      "Question text",
			Options: []models.QuestionOption{
				{Type: "text", Value: "Option A"},
				{Type: "text", Value: "Option B"},
			},
			CorrectIndex: 0,
			Difficulty:   "easy",
			CreatedAt:    time.Now(),
		})
	}

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListChapterQuestions,
	)
	w := doRequest(r, "GET", "/admin/practice/chapters/"+chapterID.Hex()+"/questions", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	questions := data["questions"].([]interface{})
	if len(questions) != 3 {
		t.Errorf("expected 3 questions, got %d", len(questions))
	}
}

// ─── AdminCreateQuestion ──────────────────────────────────────────────────────

func TestAdminCreateQuestion_InvalidChapterID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateQuestion,
	)
	w := doRequest(r, "POST", "/admin/practice/chapters/bad-id/questions", map[string]interface{}{
		"text":    "Some question?",
		"options": []map[string]string{{"type": "text", "value": "A"}, {"type": "text", "value": "B"}},
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid chapter ID, got %d", w.Code)
	}
}

func TestAdminCreateQuestion_MissingText(t *testing.T) {
	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateQuestion,
	)
	w := doRequest(r, "POST", "/admin/practice/chapters/"+chapterID.Hex()+"/questions",
		map[string]interface{}{
			"options": []map[string]string{{"type": "text", "value": "A"}, {"type": "text", "value": "B"}},
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing text, got %d", w.Code)
	}
}

func TestAdminCreateQuestion_MissingOptions(t *testing.T) {
	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateQuestion,
	)
	w := doRequest(r, "POST", "/admin/practice/chapters/"+chapterID.Hex()+"/questions",
		map[string]interface{}{
			"text": "What is 2+2?",
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing options, got %d", w.Code)
	}
}

func TestAdminCreateQuestion_TooFewOptions(t *testing.T) {
	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateQuestion,
	)
	w := doRequest(r, "POST", "/admin/practice/chapters/"+chapterID.Hex()+"/questions",
		map[string]interface{}{
			"text":    "Single option question?",
			"options": []map[string]string{{"type": "text", "value": "Only one"}},
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for fewer than 2 options, got %d", w.Code)
	}
}

func TestAdminCreateQuestion_InvalidOptionType(t *testing.T) {
	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateQuestion,
	)
	w := doRequest(r, "POST", "/admin/practice/chapters/"+chapterID.Hex()+"/questions",
		map[string]interface{}{
			"text": "Bad option type?",
			"options": []map[string]string{
				{"type": "invalid", "value": "A"},
				{"type": "text", "value": "B"},
			},
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid option type, got %d", w.Code)
	}
}

func TestAdminCreateQuestion_EmptyOptionValue(t *testing.T) {
	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateQuestion,
	)
	w := doRequest(r, "POST", "/admin/practice/chapters/"+chapterID.Hex()+"/questions",
		map[string]interface{}{
			"text": "Empty option value?",
			"options": []map[string]string{
				{"type": "text", "value": ""},
				{"type": "text", "value": "B"},
			},
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty option value, got %d", w.Code)
	}
}

func TestAdminCreateQuestion_EmptyBody(t *testing.T) {
	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateQuestion,
	)
	w := doRequest(r, "POST", "/admin/practice/chapters/"+chapterID.Hex()+"/questions", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty body, got %d", w.Code)
	}
}

func TestAdminCreateQuestion_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateQuestion,
	)
	w := doRequest(r, "POST", "/admin/practice/chapters/"+chapterID.Hex()+"/questions",
		map[string]interface{}{
			"text": "What is the capital of India?",
			"options": []map[string]string{
				{"type": "text", "value": "Mumbai"},
				{"type": "text", "value": "New Delhi"},
				{"type": "text", "value": "Kolkata"},
				{"type": "text", "value": "Chennai"},
			},
			"correctIndex": 1,
			"explanation":  "New Delhi is the capital of India.",
			"difficulty":   "easy",
		}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	question := data["question"].(map[string]interface{})
	if question["text"] != "What is the capital of India?" {
		t.Errorf("text mismatch: got %v", question["text"])
	}
}

func TestAdminCreateQuestion_DefaultsDifficulty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateQuestion,
	)
	// Omit difficulty — should default to "medium"
	w := doRequest(r, "POST", "/admin/practice/chapters/"+chapterID.Hex()+"/questions",
		map[string]interface{}{
			"text": "No difficulty specified?",
			"options": []map[string]string{
				{"type": "text", "value": "Yes"},
				{"type": "text", "value": "No"},
			},
			"correctIndex": 0,
		}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	question := data["question"].(map[string]interface{})
	if question["difficulty"] != "medium" {
		t.Errorf("difficulty: want medium (default), got %v", question["difficulty"])
	}
}

func TestAdminCreateQuestion_ImageOptionType(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	chapterID := primitive.NewObjectID()
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/practice/chapters/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminCreateQuestion,
	)
	w := doRequest(r, "POST", "/admin/practice/chapters/"+chapterID.Hex()+"/questions",
		map[string]interface{}{
			"text": "Which image is correct?",
			"options": []map[string]string{
				{"type": "image", "value": "https://example.com/img1.png"},
				{"type": "image", "value": "https://example.com/img2.png"},
			},
			"correctIndex": 0,
		}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201 for image options, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminUpdateQuestion ──────────────────────────────────────────────────────

func TestAdminUpdateQuestion_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/practice/questions/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateQuestion,
	)
	w := doRequest(r, "PUT", "/admin/practice/questions/bad-id", map[string]interface{}{
		"text": "Updated text",
		"options": []map[string]string{
			{"type": "text", "value": "A"},
			{"type": "text", "value": "B"},
		},
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid question ID, got %d", w.Code)
	}
}

func TestAdminUpdateQuestion_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	adminID := primitive.NewObjectID()
	id := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/practice/questions/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateQuestion,
	)
	w := doRequest(r, "PUT", "/admin/practice/questions/"+id.Hex(), map[string]interface{}{
		"text": "Ghost question",
		"options": []map[string]string{
			{"type": "text", "value": "A"},
			{"type": "text", "value": "B"},
		},
	}, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent question, got %d", w.Code)
	}
}

func TestAdminUpdateQuestion_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	chapterID := primitive.NewObjectID()
	cid := chapterID
	ctx := context.Background()
	question := models.Question{
		ID:        primitive.NewObjectID(),
		ChapterID: &cid,
		Text:      "Old question text",
		Options: []models.QuestionOption{
			{Type: "text", Value: "Old A"},
			{Type: "text", Value: "Old B"},
		},
		CorrectIndex: 0,
		Difficulty:   "easy",
		CreatedAt:    time.Now(),
	}
	config.GetCollection("questions").InsertOne(ctx, question)

	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/practice/questions/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminUpdateQuestion,
	)
	w := doRequest(r, "PUT", "/admin/practice/questions/"+question.ID.Hex(), map[string]interface{}{
		"text": "Updated question text",
		"options": []map[string]string{
			{"type": "text", "value": "New A"},
			{"type": "text", "value": "New B"},
			{"type": "text", "value": "New C"},
		},
		"correctIndex": 2,
		"difficulty":   "hard",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminDeleteQuestion ──────────────────────────────────────────────────────

func TestAdminDeleteQuestion_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/practice/questions/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteQuestion,
	)
	w := doRequest(r, "DELETE", "/admin/practice/questions/bad-id", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid question ID, got %d", w.Code)
	}
}

func TestAdminDeleteQuestion_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/practice/questions/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteQuestion,
	)
	w := doRequest(r, "DELETE", "/admin/practice/questions/"+primitive.NewObjectID().Hex(), nil, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent question, got %d", w.Code)
	}
}

func TestAdminDeleteQuestion_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	chapterID := primitive.NewObjectID()
	cid := chapterID
	ctx := context.Background()
	question := models.Question{
		ID:        primitive.NewObjectID(),
		ChapterID: &cid,
		Text:      "Delete this question",
		Options: []models.QuestionOption{
			{Type: "text", Value: "A"},
			{Type: "text", Value: "B"},
		},
		CorrectIndex: 0,
		CreatedAt:    time.Now(),
	}
	config.GetCollection("questions").InsertOne(ctx, question)

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/practice/questions/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteQuestion,
	)
	w := doRequest(r, "DELETE", "/admin/practice/questions/"+question.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── ListSubjects (Student) ───────────────────────────────────────────────────

func TestListSubjects_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "subjects")
	dropCollection(t, "chapters")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/practice/subjects",
		setUserID(userID.Hex(), "9876543210"),
		ListSubjects,
	)
	w := doRequest(r, "GET", "/practice/subjects", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	subjects, ok := data["subjects"].([]interface{})
	if !ok {
		t.Fatal("expected subjects array")
	}
	if len(subjects) != 0 {
		t.Errorf("expected 0 subjects, got %d", len(subjects))
	}
}

func TestListSubjects_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "subjects")
	dropCollection(t, "chapters")

	ctx := context.Background()
	for i := 1; i <= 2; i++ {
		config.GetCollection("subjects").InsertOne(ctx, models.Subject{
			ID:        primitive.NewObjectID(),
			Name:      "Subject",
			Order:     i,
			CreatedAt: time.Now(),
		})
	}

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/practice/subjects",
		setUserID(userID.Hex(), "9876543210"),
		ListSubjects,
	)
	w := doRequest(r, "GET", "/practice/subjects", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	subjects := data["subjects"].([]interface{})
	if len(subjects) != 2 {
		t.Errorf("expected 2 subjects, got %d", len(subjects))
	}
}

func TestListSubjects_IncludesChapterCount(t *testing.T) {
	requireDB(t)
	dropCollection(t, "subjects")
	dropCollection(t, "chapters")

	ctx := context.Background()
	subject := models.Subject{
		ID:        primitive.NewObjectID(),
		Name:      "Maths",
		Order:     1,
		CreatedAt: time.Now(),
	}
	config.GetCollection("subjects").InsertOne(ctx, subject)

	// Add 2 chapters for this subject
	for i := 1; i <= 2; i++ {
		sid := subject.ID
		config.GetCollection("chapters").InsertOne(ctx, models.Chapter{
			ID:        primitive.NewObjectID(),
			SubjectID: &sid,
			Title:     "Chapter",
			Order:     i,
			CreatedAt: time.Now(),
		})
	}

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/practice/subjects",
		setUserID(userID.Hex(), "9876543210"),
		ListSubjects,
	)
	w := doRequest(r, "GET", "/practice/subjects", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	subjects := data["subjects"].([]interface{})
	if len(subjects) != 1 {
		t.Fatalf("expected 1 subject, got %d", len(subjects))
	}
	s := subjects[0].(map[string]interface{})
	if s["chapterCount"].(float64) != 2 {
		t.Errorf("chapterCount: want 2, got %v", s["chapterCount"])
	}
}

// ─── ListSubjectChapters (Student) ───────────────────────────────────────────

func TestListSubjectChapters_InvalidSubjectID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("GET", "/practice/subjects/:id/chapters",
		setUserID(userID.Hex(), "9876543210"),
		ListSubjectChapters,
	)
	w := doRequest(r, "GET", "/practice/subjects/bad-id/chapters", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid subject ID, got %d", w.Code)
	}
}

func TestListSubjectChapters_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")
	dropCollection(t, "userchapterprogress")

	subjectID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	r := newRouter("GET", "/practice/subjects/:id/chapters",
		setUserID(userID.Hex(), "9876543210"),
		ListSubjectChapters,
	)
	w := doRequest(r, "GET", "/practice/subjects/"+subjectID.Hex()+"/chapters", nil, nil)

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

func TestListSubjectChapters_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")
	dropCollection(t, "questions")
	dropCollection(t, "userchapterprogress")

	subjectID := primitive.NewObjectID()
	ctx := context.Background()
	for i := 1; i <= 3; i++ {
		sid := subjectID
		config.GetCollection("chapters").InsertOne(ctx, models.Chapter{
			ID:        primitive.NewObjectID(),
			SubjectID: &sid,
			Title:     "Chapter",
			Order:     i,
			CreatedAt: time.Now(),
		})
	}

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/practice/subjects/:id/chapters",
		setUserID(userID.Hex(), "9876543210"),
		ListSubjectChapters,
	)
	w := doRequest(r, "GET", "/practice/subjects/"+subjectID.Hex()+"/chapters", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	chapters := data["chapters"].([]interface{})
	if len(chapters) != 3 {
		t.Errorf("expected 3 chapters, got %d", len(chapters))
	}
}

func TestListSubjectChapters_IncludesProgressCounts(t *testing.T) {
	requireDB(t)
	dropCollection(t, "chapters")
	dropCollection(t, "questions")
	dropCollection(t, "userchapterprogress")

	subjectID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	ctx := context.Background()

	sid := subjectID
	chapter := models.Chapter{
		ID:        primitive.NewObjectID(),
		SubjectID: &sid,
		Title:     "Chapter With Questions",
		Order:     1,
		CreatedAt: time.Now(),
	}
	config.GetCollection("chapters").InsertOne(ctx, chapter)

	// Add 2 questions to the chapter
	q1ID := primitive.NewObjectID()
	q2ID := primitive.NewObjectID()
	cid := chapter.ID
	config.GetCollection("questions").InsertOne(ctx, models.Question{
		ID: q1ID, ChapterID: &cid, Text: "Q1",
		Options:   []models.QuestionOption{{Type: "text", Value: "A"}, {Type: "text", Value: "B"}},
		CreatedAt: time.Now(),
	})
	config.GetCollection("questions").InsertOne(ctx, models.Question{
		ID: q2ID, ChapterID: &cid, Text: "Q2",
		Options:   []models.QuestionOption{{Type: "text", Value: "A"}, {Type: "text", Value: "B"}},
		CreatedAt: time.Now(),
	})

	// Mark one question as solved
	config.GetCollection("userchapterprogress").InsertOne(ctx, models.UserChapterProgress{
		ID:                primitive.NewObjectID(),
		UserID:            userID,
		ChapterID:         chapter.ID,
		SolvedQuestionIDs: []primitive.ObjectID{q1ID},
		UpdatedAt:         time.Now(),
	})

	r := newRouter("GET", "/practice/subjects/:id/chapters",
		setUserID(userID.Hex(), "9876543210"),
		ListSubjectChapters,
	)
	w := doRequest(r, "GET", "/practice/subjects/"+subjectID.Hex()+"/chapters", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	chapters := data["chapters"].([]interface{})
	if len(chapters) != 1 {
		t.Fatalf("expected 1 chapter, got %d", len(chapters))
	}
	ch := chapters[0].(map[string]interface{})
	if ch["questionCount"].(float64) != 2 {
		t.Errorf("questionCount: want 2, got %v", ch["questionCount"])
	}
	if ch["solvedCount"].(float64) != 1 {
		t.Errorf("solvedCount: want 1, got %v", ch["solvedCount"])
	}
}

// ─── GetChapterQuestions (Student) ───────────────────────────────────────────

func TestGetChapterQuestions_InvalidChapterID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("GET", "/practice/chapters/:id/questions",
		setUserID(userID.Hex(), "9876543210"),
		GetChapterQuestions,
	)
	w := doRequest(r, "GET", "/practice/chapters/bad-id/questions", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid chapter ID, got %d", w.Code)
	}
}

func TestGetChapterQuestions_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")
	dropCollection(t, "userchapterprogress")

	chapterID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	r := newRouter("GET", "/practice/chapters/:id/questions",
		setUserID(userID.Hex(), "9876543210"),
		GetChapterQuestions,
	)
	w := doRequest(r, "GET", "/practice/chapters/"+chapterID.Hex()+"/questions", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	questions, ok := data["questions"].([]interface{})
	if !ok {
		t.Fatal("expected questions array")
	}
	if len(questions) != 0 {
		t.Errorf("expected 0 questions, got %d", len(questions))
	}
}

func TestGetChapterQuestions_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")
	dropCollection(t, "userchapterprogress")

	chapterID := primitive.NewObjectID()
	ctx := context.Background()
	for i := 0; i < 4; i++ {
		cid := chapterID
		config.GetCollection("questions").InsertOne(ctx, models.Question{
			ID:        primitive.NewObjectID(),
			ChapterID: &cid,
			Text:      "Question",
			Options: []models.QuestionOption{
				{Type: "text", Value: "A"},
				{Type: "text", Value: "B"},
			},
			CorrectIndex: 0,
			Difficulty:   "medium",
			CreatedAt:    time.Now(),
		})
	}

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/practice/chapters/:id/questions",
		setUserID(userID.Hex(), "9876543210"),
		GetChapterQuestions,
	)
	w := doRequest(r, "GET", "/practice/chapters/"+chapterID.Hex()+"/questions", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	questions := data["questions"].([]interface{})
	if len(questions) != 4 {
		t.Errorf("expected 4 questions, got %d", len(questions))
	}
}

func TestGetChapterQuestions_IncludesSolvedIDs(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")
	dropCollection(t, "userchapterprogress")

	chapterID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	ctx := context.Background()

	cid := chapterID
	q := models.Question{
		ID:        primitive.NewObjectID(),
		ChapterID: &cid,
		Text:      "Solved question",
		Options: []models.QuestionOption{
			{Type: "text", Value: "A"},
			{Type: "text", Value: "B"},
		},
		CorrectIndex: 0,
		CreatedAt:    time.Now(),
	}
	config.GetCollection("questions").InsertOne(ctx, q)

	config.GetCollection("userchapterprogress").InsertOne(ctx, models.UserChapterProgress{
		ID:                primitive.NewObjectID(),
		UserID:            userID,
		ChapterID:         chapterID,
		SolvedQuestionIDs: []primitive.ObjectID{q.ID},
		UpdatedAt:         time.Now(),
	})

	r := newRouter("GET", "/practice/chapters/:id/questions",
		setUserID(userID.Hex(), "9876543210"),
		GetChapterQuestions,
	)
	w := doRequest(r, "GET", "/practice/chapters/"+chapterID.Hex()+"/questions", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	solvedIDs, ok := data["solvedIds"].([]interface{})
	if !ok {
		t.Fatal("expected solvedIds array")
	}
	if len(solvedIDs) != 1 {
		t.Errorf("expected 1 solved ID, got %d", len(solvedIDs))
	}
	if solvedIDs[0] != q.ID.Hex() {
		t.Errorf("solvedId: want %s, got %v", q.ID.Hex(), solvedIDs[0])
	}
}

func TestGetChapterQuestions_PYQFilter(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")
	dropCollection(t, "userchapterprogress")

	chapterID := primitive.NewObjectID()
	ctx := context.Background()
	cid := chapterID

	// Insert 1 PYQ and 1 regular question
	config.GetCollection("questions").InsertOne(ctx, models.Question{
		ID:        primitive.NewObjectID(),
		ChapterID: &cid,
		Text:      "PYQ question",
		Options: []models.QuestionOption{
			{Type: "text", Value: "A"},
			{Type: "text", Value: "B"},
		},
		IsPYQ:     true,
		CreatedAt: time.Now(),
	})
	config.GetCollection("questions").InsertOne(ctx, models.Question{
		ID:        primitive.NewObjectID(),
		ChapterID: &cid,
		Text:      "Normal question",
		Options: []models.QuestionOption{
			{Type: "text", Value: "A"},
			{Type: "text", Value: "B"},
		},
		IsPYQ:     false,
		CreatedAt: time.Now(),
	})

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/practice/chapters/:id/questions",
		setUserID(userID.Hex(), "9876543210"),
		GetChapterQuestions,
	)
	w := doRequest(r, "GET", "/practice/chapters/"+chapterID.Hex()+"/questions?pyq=true", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	questions := data["questions"].([]interface{})
	if len(questions) != 1 {
		t.Errorf("expected 1 PYQ question, got %d", len(questions))
	}
}

// ─── SubmitChapterPractice (Student) ─────────────────────────────────────────

func TestSubmitChapterPractice_InvalidChapterID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/practice/chapters/:id/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitChapterPractice,
	)
	w := doRequest(r, "POST", "/practice/chapters/bad-id/submit", map[string]interface{}{
		"answers": map[string]int{},
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid chapter ID, got %d", w.Code)
	}
}

func TestSubmitChapterPractice_MissingAnswers(t *testing.T) {
	chapterID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/practice/chapters/:id/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitChapterPractice,
	)
	w := doRequest(r, "POST", "/practice/chapters/"+chapterID.Hex()+"/submit",
		map[string]interface{}{
			// missing "answers" field
		}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing answers, got %d", w.Code)
	}
}

func TestSubmitChapterPractice_EmptyBody(t *testing.T) {
	chapterID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/practice/chapters/:id/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitChapterPractice,
	)
	// nil body → EOF → 400
	w := doRequest(r, "POST", "/practice/chapters/"+chapterID.Hex()+"/submit", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty body, got %d", w.Code)
	}
}

func TestSubmitChapterPractice_EmptyAnswersMap(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")
	dropCollection(t, "userchapterprogress")

	chapterID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/practice/chapters/:id/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitChapterPractice,
	)
	w := doRequest(r, "POST", "/practice/chapters/"+chapterID.Hex()+"/submit",
		map[string]interface{}{
			"answers": map[string]int{},
		}, nil)

	// No questions found → total=0, correct=0, percent=0
	if w.Code != http.StatusOK {
		t.Errorf("want 200 for empty answers, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	result := data["result"].(map[string]interface{})
	if result["total"].(float64) != 0 {
		t.Errorf("total: want 0, got %v", result["total"])
	}
}

func TestSubmitChapterPractice_AllCorrect(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")
	dropCollection(t, "userchapterprogress")

	chapterID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	ctx := context.Background()

	cid := chapterID
	q1 := models.Question{
		ID:        primitive.NewObjectID(),
		ChapterID: &cid,
		Text:      "Q1",
		Options: []models.QuestionOption{
			{Type: "text", Value: "Wrong"},
			{Type: "text", Value: "Right"},
		},
		CorrectIndex: 1,
		Difficulty:   "easy",
		CreatedAt:    time.Now(),
	}
	q2 := models.Question{
		ID:        primitive.NewObjectID(),
		ChapterID: &cid,
		Text:      "Q2",
		Options: []models.QuestionOption{
			{Type: "text", Value: "Right"},
			{Type: "text", Value: "Wrong"},
		},
		CorrectIndex: 0,
		Difficulty:   "medium",
		CreatedAt:    time.Now(),
	}
	config.GetCollection("questions").InsertOne(ctx, q1)
	config.GetCollection("questions").InsertOne(ctx, q2)

	r := newRouter("POST", "/practice/chapters/:id/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitChapterPractice,
	)
	w := doRequest(r, "POST", "/practice/chapters/"+chapterID.Hex()+"/submit",
		map[string]interface{}{
			"answers": map[string]int{
				q1.ID.Hex(): 1, // correct
				q2.ID.Hex(): 0, // correct
			},
		}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	result := data["result"].(map[string]interface{})
	if result["correct"].(float64) != 2 {
		t.Errorf("correct: want 2, got %v", result["correct"])
	}
	if result["total"].(float64) != 2 {
		t.Errorf("total: want 2, got %v", result["total"])
	}
	if result["percent"].(float64) != 100 {
		t.Errorf("percent: want 100, got %v", result["percent"])
	}
}

func TestSubmitChapterPractice_AllWrong(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")
	dropCollection(t, "userchapterprogress")

	chapterID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	ctx := context.Background()

	cid := chapterID
	q := models.Question{
		ID:        primitive.NewObjectID(),
		ChapterID: &cid,
		Text:      "Q1",
		Options: []models.QuestionOption{
			{Type: "text", Value: "A"},
			{Type: "text", Value: "B"},
		},
		CorrectIndex: 1,
		CreatedAt:    time.Now(),
	}
	config.GetCollection("questions").InsertOne(ctx, q)

	r := newRouter("POST", "/practice/chapters/:id/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitChapterPractice,
	)
	w := doRequest(r, "POST", "/practice/chapters/"+chapterID.Hex()+"/submit",
		map[string]interface{}{
			"answers": map[string]int{
				q.ID.Hex(): 0, // wrong (correct is 1)
			},
		}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	result := data["result"].(map[string]interface{})
	if result["correct"].(float64) != 0 {
		t.Errorf("correct: want 0, got %v", result["correct"])
	}
	if result["percent"].(float64) != 0 {
		t.Errorf("percent: want 0, got %v", result["percent"])
	}
}

func TestSubmitChapterPractice_PartialCorrect(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")
	dropCollection(t, "userchapterprogress")

	chapterID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	ctx := context.Background()

	cid := chapterID
	q1 := models.Question{
		ID:           primitive.NewObjectID(),
		ChapterID:    &cid,
		Text:         "Q1",
		Options:      []models.QuestionOption{{Type: "text", Value: "A"}, {Type: "text", Value: "B"}},
		CorrectIndex: 0,
		CreatedAt:    time.Now(),
	}
	q2 := models.Question{
		ID:           primitive.NewObjectID(),
		ChapterID:    &cid,
		Text:         "Q2",
		Options:      []models.QuestionOption{{Type: "text", Value: "A"}, {Type: "text", Value: "B"}},
		CorrectIndex: 1,
		CreatedAt:    time.Now(),
	}
	config.GetCollection("questions").InsertOne(ctx, q1)
	config.GetCollection("questions").InsertOne(ctx, q2)

	r := newRouter("POST", "/practice/chapters/:id/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitChapterPractice,
	)
	w := doRequest(r, "POST", "/practice/chapters/"+chapterID.Hex()+"/submit",
		map[string]interface{}{
			"answers": map[string]int{
				q1.ID.Hex(): 0, // correct
				q2.ID.Hex(): 0, // wrong
			},
		}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	result := data["result"].(map[string]interface{})
	if result["correct"].(float64) != 1 {
		t.Errorf("correct: want 1, got %v", result["correct"])
	}
	if result["percent"].(float64) != 50 {
		t.Errorf("percent: want 50, got %v", result["percent"])
	}
	detailed, ok := result["detailed"].([]interface{})
	if !ok {
		t.Fatal("expected detailed array")
	}
	if len(detailed) != 2 {
		t.Errorf("expected 2 detailed items, got %d", len(detailed))
	}
}

func TestSubmitChapterPractice_UpdatesProgress(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")
	dropCollection(t, "userchapterprogress")

	chapterID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	ctx := context.Background()

	cid := chapterID
	q := models.Question{
		ID:           primitive.NewObjectID(),
		ChapterID:    &cid,
		Text:         "Progress Q",
		Options:      []models.QuestionOption{{Type: "text", Value: "A"}, {Type: "text", Value: "B"}},
		CorrectIndex: 0,
		CreatedAt:    time.Now(),
	}
	config.GetCollection("questions").InsertOne(ctx, q)

	r := newRouter("POST", "/practice/chapters/:id/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitChapterPractice,
	)
	w := doRequest(r, "POST", "/practice/chapters/"+chapterID.Hex()+"/submit",
		map[string]interface{}{
			"answers": map[string]int{q.ID.Hex(): 0},
		}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify progress was upserted in DB
	var progress models.UserChapterProgress
	err := config.GetCollection("userchapterprogress").FindOne(ctx, map[string]interface{}{
		"userId":    userID,
		"chapterId": chapterID,
	}).Decode(&progress)
	if err != nil {
		t.Errorf("expected userchapterprogress record to exist: %v", err)
	}
	if len(progress.SolvedQuestionIDs) != 1 {
		t.Errorf("expected 1 solved question ID in progress, got %d", len(progress.SolvedQuestionIDs))
	}
}
