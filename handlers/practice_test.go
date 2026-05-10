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

// ─── GetPracticeQuestions ─────────────────────────────────────────────────────

func TestGetPracticeQuestions_InvalidChapterID(t *testing.T) {
	r := newRouter("GET", "/practice/:chapterId/questions", GetPracticeQuestions)
	w := doRequest(r, "GET", "/practice/bad-id/questions", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestGetPracticeQuestions_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	chapterID := primitive.NewObjectID()
	r := newRouter("GET", "/practice/:chapterId/questions", GetPracticeQuestions)
	w := doRequest(r, "GET", "/practice/"+chapterID.Hex()+"/questions", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	qs, ok := data["questions"].([]interface{})
	if !ok {
		t.Fatal("expected questions array")
	}
	if len(qs) != 0 {
		t.Errorf("expected 0 questions, got %d", len(qs))
	}
}

func TestGetPracticeQuestions_WithQuestions(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	chapterID := primitive.NewObjectID()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		cid := chapterID
		config.GetCollection("questions").InsertOne(ctx, models.Question{
			ID:        primitive.NewObjectID(),
			ChapterID: &cid,
			Text:      "Q",
			Options: []models.QuestionOption{
				{Type: "text", Value: "A"},
				{Type: "text", Value: "B"},
			},
			CorrectIndex: 0,
			CreatedAt:    time.Now(),
		})
	}

	r := newRouter("GET", "/practice/:chapterId/questions", GetPracticeQuestions)
	w := doRequest(r, "GET", "/practice/"+chapterID.Hex()+"/questions", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	qs := data["questions"].([]interface{})
	if len(qs) != 3 {
		t.Errorf("expected 3 questions, got %d", len(qs))
	}
}

// ─── SubmitPractice ───────────────────────────────────────────────────────────

func TestSubmitPractice_MissingFields(t *testing.T) {
	r := newRouter("POST", "/practice/submit", SubmitPractice)
	w := doRequest(r, "POST", "/practice/submit", map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing fields, got %d", w.Code)
	}
}

func TestSubmitPractice_InvalidChapterID(t *testing.T) {
	r := newRouter("POST", "/practice/submit", SubmitPractice)
	w := doRequest(r, "POST", "/practice/submit", map[string]interface{}{
		"chapterId": "bad-id",
		"answers":   map[string]int{"0": 0},
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid chapterId, got %d", w.Code)
	}
}

func TestSubmitPractice_NoQuestions(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	chapterID := primitive.NewObjectID()
	r := newRouter("POST", "/practice/submit", SubmitPractice)
	w := doRequest(r, "POST", "/practice/submit", map[string]interface{}{
		"chapterId": chapterID.Hex(),
		"answers":   map[string]int{},
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200 (0 questions = 100%%), got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	result := data["result"].(map[string]interface{})
	if result["total"].(float64) != 0 {
		t.Errorf("total: want 0, got %v", result["total"])
	}
}

func TestSubmitPractice_WithCorrectAnswers(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	chapterID := primitive.NewObjectID()
	ctx := context.Background()
	cid := chapterID
	config.GetCollection("questions").InsertOne(ctx, models.Question{
		ID:        primitive.NewObjectID(),
		ChapterID: &cid,
		Text:      "What is 2+2?",
		Options: []models.QuestionOption{
			{Type: "text", Value: "3"},
			{Type: "text", Value: "4"},
		},
		CorrectIndex: 1,
		CreatedAt:    time.Now(),
	})

	r := newRouter("POST", "/practice/submit", SubmitPractice)
	w := doRequest(r, "POST", "/practice/submit", map[string]interface{}{
		"chapterId": chapterID.Hex(),
		"answers":   map[string]int{"0": 1}, // correct index
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	result := data["result"].(map[string]interface{})
	if result["correct"].(float64) != 1 {
		t.Errorf("correct: want 1, got %v", result["correct"])
	}
	if result["percent"].(float64) != 100 {
		t.Errorf("percent: want 100, got %v", result["percent"])
	}
}
