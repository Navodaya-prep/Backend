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

func insertMockTest(t *testing.T, title string) models.MockTest {
	t.Helper()
	test := models.MockTest{
		ID:          primitive.NewObjectID(),
		Title:       title,
		Subject:     "Maths",
		Duration:    60,
		ClassLevel:  "6",
		QuestionIDs: []primitive.ObjectID{},
		CreatedAt:   time.Now(),
	}
	config.GetCollection("mocktests").InsertOne(context.Background(), test)
	return test
}

// ─── ListAdminMockTests ───────────────────────────────────────────────────────

func TestListAdminMockTests_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	superID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/mocktests",
		setAdminID(superID.Hex(), "super@test.com", true),
		ListAdminMockTests,
	)
	w := doRequest(r, "GET", "/admin/mocktests", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	tests, ok := data["tests"].([]interface{})
	if !ok {
		t.Fatal("expected tests array")
	}
	if len(tests) != 0 {
		t.Errorf("expected 0 tests, got %d", len(tests))
	}
}

func TestListAdminMockTests_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	insertMockTest(t, "Test 1")
	insertMockTest(t, "Test 2")

	superID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/mocktests",
		setAdminID(superID.Hex(), "super@test.com", true),
		ListAdminMockTests,
	)
	w := doRequest(r, "GET", "/admin/mocktests", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	tests := data["tests"].([]interface{})
	if len(tests) != 2 {
		t.Errorf("expected 2 tests, got %d", len(tests))
	}
}

// ─── ListAdminMockTestQuestions ───────────────────────────────────────────────

func TestListAdminMockTestQuestions_InvalidID(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/mocktests/:id/questions",
		setAdminID(superID.Hex(), "super@test.com", true),
		ListAdminMockTestQuestions,
	)
	w := doRequest(r, "GET", "/admin/mocktests/bad-id/questions", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestListAdminMockTestQuestions_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/mocktests/:id/questions",
		setAdminID(superID.Hex(), "super@test.com", true),
		ListAdminMockTestQuestions,
	)
	w := doRequest(r, "GET", "/admin/mocktests/"+testID.Hex()+"/questions", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestListAdminMockTestQuestions_NoQuestions(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	test := insertMockTest(t, "Empty Test")

	superID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/mocktests/:id/questions",
		setAdminID(superID.Hex(), "super@test.com", true),
		ListAdminMockTestQuestions,
	)
	w := doRequest(r, "GET", "/admin/mocktests/"+test.ID.Hex()+"/questions", nil, nil)

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

func TestListAdminMockTestQuestions_WithQuestions(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "questions")

	q := models.Question{
		ID:      primitive.NewObjectID(),
		Text:    "What is Go?",
		Subject: "CS",
		Options: []models.QuestionOption{
			{Type: "text", Value: "A language"},
			{Type: "text", Value: "A country"},
		},
		CorrectIndex: 0,
		CreatedAt:    time.Now(),
	}
	config.GetCollection("questions").InsertOne(context.Background(), q)

	test := models.MockTest{
		ID:          primitive.NewObjectID(),
		Title:       "Test With Questions",
		Subject:     "CS",
		Duration:    30,
		ClassLevel:  "9",
		QuestionIDs: []primitive.ObjectID{q.ID},
		CreatedAt:   time.Now(),
	}
	config.GetCollection("mocktests").InsertOne(context.Background(), test)

	superID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/mocktests/:id/questions",
		setAdminID(superID.Hex(), "super@test.com", true),
		ListAdminMockTestQuestions,
	)
	w := doRequest(r, "GET", "/admin/mocktests/"+test.ID.Hex()+"/questions", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	qs := data["questions"].([]interface{})
	if len(qs) != 1 {
		t.Errorf("expected 1 question, got %d", len(qs))
	}
}

// ─── CreateMockTest ───────────────────────────────────────────────────────────

func TestCreateMockTest_MissingFields(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/mocktests",
		setAdminID(superID.Hex(), "super@test.com", true),
		CreateMockTest,
	)
	w := doRequest(r, "POST", "/admin/mocktests", map[string]interface{}{
		"title": "Only Title",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing fields, got %d", w.Code)
	}
}

func TestCreateMockTest_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/mocktests",
		setAdminID(superID.Hex(), "super@test.com", true),
		CreateMockTest,
	)
	w := doRequest(r, "POST", "/admin/mocktests", map[string]interface{}{
		"title":      "New Test",
		"subject":    "Science",
		"duration":   45,
		"classLevel": "7",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	test := data["test"].(map[string]interface{})
	if test["title"] != "New Test" {
		t.Errorf("title: want New Test, got %v", test["title"])
	}
}

func TestCreateMockTest_WithPremiumFlag(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/mocktests",
		setAdminID(superID.Hex(), "super@test.com", true),
		CreateMockTest,
	)
	w := doRequest(r, "POST", "/admin/mocktests", map[string]interface{}{
		"title":      "Premium Test",
		"subject":    "English",
		"duration":   30,
		"classLevel": "8",
		"isPremium":  true,
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", w.Code)
	}
}

// ─── UpdateMockTest ───────────────────────────────────────────────────────────

func TestUpdateMockTest_InvalidID(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		UpdateMockTest,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/bad-id", map[string]interface{}{
		"title": "T", "subject": "S", "duration": 10, "classLevel": "6",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestUpdateMockTest_MissingFields(t *testing.T) {
	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		UpdateMockTest,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/"+testID.Hex(), map[string]interface{}{
		"title": "Only",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing fields, got %d", w.Code)
	}
}

func TestUpdateMockTest_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		UpdateMockTest,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/"+testID.Hex(), map[string]interface{}{
		"title": "T", "subject": "S", "duration": 10, "classLevel": "6",
	}, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestUpdateMockTest_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	test := insertMockTest(t, "Old Title")

	superID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		UpdateMockTest,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/"+test.ID.Hex(), map[string]interface{}{
		"title":      "New Title",
		"subject":    "Science",
		"duration":   90,
		"classLevel": "9",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AddQuestionToMockTest ────────────────────────────────────────────────────

func TestAddQuestionToMockTest_InvalidTestID(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/mocktests/:id/questions",
		setAdminID(superID.Hex(), "super@test.com", true),
		AddQuestionToMockTest,
	)
	w := doRequest(r, "POST", "/admin/mocktests/bad-id/questions", map[string]interface{}{
		"text":    "Q",
		"options": []map[string]string{{"type": "text", "value": "A"}, {"type": "text", "value": "B"}},
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid test ID, got %d", w.Code)
	}
}

func TestAddQuestionToMockTest_MissingText(t *testing.T) {
	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/mocktests/:id/questions",
		setAdminID(superID.Hex(), "super@test.com", true),
		AddQuestionToMockTest,
	)
	w := doRequest(r, "POST", "/admin/mocktests/"+testID.Hex()+"/questions", map[string]interface{}{
		"options": []map[string]string{{"type": "text", "value": "A"}},
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing text, got %d", w.Code)
	}
}

func TestAddQuestionToMockTest_TooFewOptions(t *testing.T) {
	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/mocktests/:id/questions",
		setAdminID(superID.Hex(), "super@test.com", true),
		AddQuestionToMockTest,
	)
	w := doRequest(r, "POST", "/admin/mocktests/"+testID.Hex()+"/questions", map[string]interface{}{
		"text":    "Q",
		"options": []map[string]string{{"type": "text", "value": "A"}}, // only 1 option
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for <2 options, got %d", w.Code)
	}
}

func TestAddQuestionToMockTest_InvalidCorrectIndex(t *testing.T) {
	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/mocktests/:id/questions",
		setAdminID(superID.Hex(), "super@test.com", true),
		AddQuestionToMockTest,
	)
	w := doRequest(r, "POST", "/admin/mocktests/"+testID.Hex()+"/questions", map[string]interface{}{
		"text":         "Q",
		"options":      []map[string]string{{"type": "text", "value": "A"}, {"type": "text", "value": "B"}},
		"correctIndex": 5, // out of range
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for out-of-range correctIndex, got %d", w.Code)
	}
}

func TestAddQuestionToMockTest_TestNotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/mocktests/:id/questions",
		setAdminID(superID.Hex(), "super@test.com", true),
		AddQuestionToMockTest,
	)
	w := doRequest(r, "POST", "/admin/mocktests/"+testID.Hex()+"/questions", map[string]interface{}{
		"text":    "Q",
		"options": []map[string]string{{"type": "text", "value": "A"}, {"type": "text", "value": "B"}},
	}, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent test, got %d", w.Code)
	}
}

func TestAddQuestionToMockTest_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "questions")

	test := insertMockTest(t, "With Question")

	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/mocktests/:id/questions",
		setAdminID(superID.Hex(), "super@test.com", true),
		AddQuestionToMockTest,
	)
	w := doRequest(r, "POST", "/admin/mocktests/"+test.ID.Hex()+"/questions", map[string]interface{}{
		"text":         "What is 2+2?",
		"options":      []map[string]string{{"type": "text", "value": "3"}, {"type": "text", "value": "4"}},
		"correctIndex": 1,
		"subject":      "Maths",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── DeleteMockTest ───────────────────────────────────────────────────────────

func TestDeleteMockTest_InvalidID(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/mocktests/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteMockTest,
	)
	w := doRequest(r, "DELETE", "/admin/mocktests/bad-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestDeleteMockTest_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/mocktests/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteMockTest,
	)
	w := doRequest(r, "DELETE", "/admin/mocktests/"+testID.Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestDeleteMockTest_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	test := insertMockTest(t, "To Delete")

	superID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/mocktests/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteMockTest,
	)
	w := doRequest(r, "DELETE", "/admin/mocktests/"+test.ID.Hex(), nil, nil)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── UpdateMockTestQuestion ───────────────────────────────────────────────────

func TestUpdateMockTestQuestion_InvalidQuestionID(t *testing.T) {
	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id/questions/:questionId",
		setAdminID(superID.Hex(), "super@test.com", true),
		UpdateMockTestQuestion,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/"+testID.Hex()+"/questions/bad-q-id", map[string]interface{}{
		"text":    "Q",
		"options": []map[string]string{{"type": "text", "value": "A"}, {"type": "text", "value": "B"}},
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestUpdateMockTestQuestion_MissingFields(t *testing.T) {
	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	questionID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id/questions/:questionId",
		setAdminID(superID.Hex(), "super@test.com", true),
		UpdateMockTestQuestion,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/"+testID.Hex()+"/questions/"+questionID.Hex(),
		map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing text/options, got %d", w.Code)
	}
}

func TestUpdateMockTestQuestion_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "questions")

	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	questionID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id/questions/:questionId",
		setAdminID(superID.Hex(), "super@test.com", true),
		UpdateMockTestQuestion,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/"+testID.Hex()+"/questions/"+questionID.Hex(),
		map[string]interface{}{
			"text":    "Updated Q",
			"options": []map[string]string{{"type": "text", "value": "A"}, {"type": "text", "value": "B"}},
		}, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

// ─── DeleteMockTestQuestion ───────────────────────────────────────────────────

func TestDeleteMockTestQuestion_InvalidTestID(t *testing.T) {
	superID := primitive.NewObjectID()
	questionID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/mocktests/:id/questions/:questionId",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteMockTestQuestion,
	)
	w := doRequest(r, "DELETE", "/admin/mocktests/bad-id/questions/"+questionID.Hex(), nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid test ID, got %d", w.Code)
	}
}

func TestDeleteMockTestQuestion_InvalidQuestionID(t *testing.T) {
	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/mocktests/:id/questions/:questionId",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteMockTestQuestion,
	)
	w := doRequest(r, "DELETE", "/admin/mocktests/"+testID.Hex()+"/questions/bad-q-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid question ID, got %d", w.Code)
	}
}

func TestDeleteMockTestQuestion_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "questions")

	questionID := primitive.NewObjectID()
	test := models.MockTest{
		ID:          primitive.NewObjectID(),
		Title:       "Test",
		Subject:     "Maths",
		Duration:    60,
		ClassLevel:  "6",
		QuestionIDs: []primitive.ObjectID{questionID},
		CreatedAt:   time.Now(),
	}
	ctx := context.Background()
	config.GetCollection("mocktests").InsertOne(ctx, test)
	config.GetCollection("questions").InsertOne(ctx, models.Question{
		ID:   questionID,
		Text: "Sample Q",
	})

	superID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/mocktests/:id/questions/:questionId",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteMockTestQuestion,
	)
	w := doRequest(r, "DELETE", "/admin/mocktests/"+test.ID.Hex()+"/questions/"+questionID.Hex(), nil, nil)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── ReorderMockTestQuestions ─────────────────────────────────────────────────

func TestReorderMockTestQuestions_InvalidTestID(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id/questions/reorder",
		setAdminID(superID.Hex(), "super@test.com", true),
		ReorderMockTestQuestions,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/bad-id/questions/reorder", map[string]interface{}{
		"questionIds": []string{},
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestReorderMockTestQuestions_MissingBody(t *testing.T) {
	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id/questions/reorder",
		setAdminID(superID.Hex(), "super@test.com", true),
		ReorderMockTestQuestions,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/"+testID.Hex()+"/questions/reorder",
		map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing questionIds, got %d", w.Code)
	}
}

func TestReorderMockTestQuestions_InvalidQuestionID(t *testing.T) {
	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id/questions/reorder",
		setAdminID(superID.Hex(), "super@test.com", true),
		ReorderMockTestQuestions,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/"+testID.Hex()+"/questions/reorder",
		map[string]interface{}{
			"questionIds": []string{"not-a-valid-id"},
		}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid question ID in array, got %d", w.Code)
	}
}

func TestReorderMockTestQuestions_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	superID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	q1 := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id/questions/reorder",
		setAdminID(superID.Hex(), "super@test.com", true),
		ReorderMockTestQuestions,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/"+testID.Hex()+"/questions/reorder",
		map[string]interface{}{
			"questionIds": []string{q1.Hex()},
		}, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent test, got %d", w.Code)
	}
}

func TestReorderMockTestQuestions_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	q1 := primitive.NewObjectID()
	q2 := primitive.NewObjectID()
	test := models.MockTest{
		ID:          primitive.NewObjectID(),
		Title:       "Reorder Test",
		Subject:     "Maths",
		Duration:    60,
		ClassLevel:  "6",
		QuestionIDs: []primitive.ObjectID{q1, q2},
		CreatedAt:   time.Now(),
	}
	config.GetCollection("mocktests").InsertOne(context.Background(), test)

	superID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/mocktests/:id/questions/reorder",
		setAdminID(superID.Hex(), "super@test.com", true),
		ReorderMockTestQuestions,
	)
	w := doRequest(r, "PUT", "/admin/mocktests/"+test.ID.Hex()+"/questions/reorder",
		map[string]interface{}{
			"questionIds": []string{q2.Hex(), q1.Hex()}, // reversed order
		}, nil)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}
