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

// ─── ListMockTests ────────────────────────────────────────────────────────────

func TestListMockTests_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "mocktestsattempts")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/mocktests", setUserID(userID.Hex(), "9876543210"), ListMockTests)
	w := doRequest(r, "GET", "/mocktests", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	tests, ok := data["tests"].([]interface{})
	if !ok {
		t.Fatal("expected tests array in response")
	}
	if len(tests) != 0 {
		t.Errorf("expected 0 tests, got %d", len(tests))
	}
}

func TestListMockTests_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "mocktestsattempts")

	insertMockTest(t, "Math Test")
	insertMockTest(t, "Science Test")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/mocktests", setUserID(userID.Hex(), "9876543210"), ListMockTests)
	w := doRequest(r, "GET", "/mocktests", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	tests := data["tests"].([]interface{})
	if len(tests) != 2 {
		t.Errorf("expected 2 tests, got %d", len(tests))
	}
}

func TestListMockTests_FilterBySubject(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "mocktestsattempts")

	ctx := context.Background()
	config.GetCollection("mocktests").InsertOne(ctx, models.MockTest{
		ID: primitive.NewObjectID(), Title: "Physics Test",
		Subject: "Physics", Duration: 30, CreatedAt: time.Now(),
	})
	insertMockTest(t, "Maths Test") // Subject="Maths" per insertMockTest helper

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/mocktests", setUserID(userID.Hex(), "9876543210"), ListMockTests)
	w := doRequest(r, "GET", "/mocktests?subject=Physics", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	tests := data["tests"].([]interface{})
	if len(tests) != 1 {
		t.Errorf("expected 1 Physics test, got %d", len(tests))
	}
}

func TestListMockTests_FilterByClass(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "mocktestsattempts")

	ctx := context.Background()
	config.GetCollection("mocktests").InsertOne(ctx, models.MockTest{
		ID: primitive.NewObjectID(), Title: "Class 10 Only",
		Subject: "Maths", ClassLevel: "10", Duration: 30, CreatedAt: time.Now(),
	})
	config.GetCollection("mocktests").InsertOne(ctx, models.MockTest{
		ID: primitive.NewObjectID(), Title: "Class 12 Only",
		Subject: "Maths", ClassLevel: "12", Duration: 30, CreatedAt: time.Now(),
	})
	config.GetCollection("mocktests").InsertOne(ctx, models.MockTest{
		ID: primitive.NewObjectID(), Title: "Both Classes",
		Subject: "Maths", ClassLevel: "both", Duration: 30, CreatedAt: time.Now(),
	})

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/mocktests", setUserID(userID.Hex(), "9876543210"), ListMockTests)
	w := doRequest(r, "GET", "/mocktests?class=10", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	tests := data["tests"].([]interface{})
	// class=10 matches "10" and "both" — expect 2
	if len(tests) != 2 {
		t.Errorf("expected 2 tests for class=10 (including 'both'), got %d", len(tests))
	}
}

func TestListMockTests_ReturnsOKWithFilters(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/mocktests", setUserID(userID.Hex(), "9876543210"), ListMockTests)
	w := doRequest(r, "GET", "/mocktests?subject=Maths&class=6", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

// ─── GetMockTest ──────────────────────────────────────────────────────────────

func TestGetMockTest_InvalidID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("GET", "/mocktests/:id", setUserID(userID.Hex(), "9876543210"), GetMockTest)
	w := doRequest(r, "GET", "/mocktests/bad-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestGetMockTest_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	userID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("GET", "/mocktests/:id", setUserID(userID.Hex(), "9876543210"), GetMockTest)
	w := doRequest(r, "GET", "/mocktests/"+testID.Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestGetMockTest_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	test := insertMockTest(t, "Full Test")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/mocktests/:id", setUserID(userID.Hex(), "9876543210"), GetMockTest)
	w := doRequest(r, "GET", "/mocktests/"+test.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	testObj, ok := data["test"].(map[string]interface{})
	if !ok {
		t.Fatal("expected test object in response")
	}
	if testObj["title"] != "Full Test" {
		t.Errorf("title: want 'Full Test', got %v", testObj["title"])
	}
}

// ─── SubmitMockTest ───────────────────────────────────────────────────────────

func TestSubmitMockTest_InvalidID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/mocktests/:id/submit", setUserID(userID.Hex(), "9876543210"), SubmitMockTest)
	w := doRequest(r, "POST", "/mocktests/bad-id/submit", map[string]interface{}{
		"answers": map[string]int{},
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestSubmitMockTest_NilBody(t *testing.T) {
	// nil body → EOF → ShouldBindJSON fails → 400
	userID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("POST", "/mocktests/:id/submit", setUserID(userID.Hex(), "9876543210"), SubmitMockTest)
	w := doRequest(r, "POST", "/mocktests/"+testID.Hex()+"/submit", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for nil body, got %d", w.Code)
	}
}

func TestSubmitMockTest_MissingAnswers(t *testing.T) {
	userID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("POST", "/mocktests/:id/submit", setUserID(userID.Hex(), "9876543210"), SubmitMockTest)
	// "answers" key absent — binding:"required" fires
	w := doRequest(r, "POST", "/mocktests/"+testID.Hex()+"/submit", map[string]interface{}{
		"timeTaken": 300,
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing answers, got %d", w.Code)
	}
}

func TestSubmitMockTest_TestNotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")

	userID := primitive.NewObjectID()
	testID := primitive.NewObjectID()
	r := newRouter("POST", "/mocktests/:id/submit", setUserID(userID.Hex(), "9876543210"), SubmitMockTest)
	w := doRequest(r, "POST", "/mocktests/"+testID.Hex()+"/submit", map[string]interface{}{
		"answers": map[string]int{},
	}, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent test, got %d", w.Code)
	}
}

func TestSubmitMockTest_Success_NoQuestions(t *testing.T) {
	// Insert a test with no questions, submit with empty answers → score=0, totalMarks=0, 200 OK
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "mocktestsattempts")

	test := insertMockTest(t, "Empty Test") // QuestionIDs is empty per helper

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/mocktests/:id/submit", setUserID(userID.Hex(), "9876543210"), SubmitMockTest)
	w := doRequest(r, "POST", "/mocktests/"+test.ID.Hex()+"/submit", map[string]interface{}{
		"answers":   map[string]int{},
		"timeTaken": 120,
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	result := data["result"].(map[string]interface{})
	if result["score"].(float64) != 0 {
		t.Errorf("score: want 0, got %v", result["score"])
	}
	if result["totalMarks"].(float64) != 0 {
		t.Errorf("totalMarks: want 0, got %v", result["totalMarks"])
	}
}

func TestSubmitMockTest_ResultFieldsPresent(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "mocktestsattempts")

	test := insertMockTest(t, "Fields Test")

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/mocktests/:id/submit", setUserID(userID.Hex(), "9876543210"), SubmitMockTest)
	w := doRequest(r, "POST", "/mocktests/"+test.ID.Hex()+"/submit", map[string]interface{}{
		"answers":   map[string]int{},
		"timeTaken": 90,
	}, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	result := data["result"].(map[string]interface{})

	requiredFields := []string{"attemptId", "score", "totalMarks", "correct", "wrong", "percent", "timeTaken", "detailed"}
	for _, field := range requiredFields {
		if _, ok := result[field]; !ok {
			t.Errorf("missing field %q in submit result", field)
		}
	}
	if result["timeTaken"].(float64) != 90 {
		t.Errorf("timeTaken: want 90, got %v", result["timeTaken"])
	}
}

func TestSubmitMockTest_SavesAttempt(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "mocktestsattempts")

	test := insertMockTest(t, "Save Attempt Test")
	userID := primitive.NewObjectID()

	r := newRouter("POST", "/mocktests/:id/submit", setUserID(userID.Hex(), "9876543210"), SubmitMockTest)
	w := doRequest(r, "POST", "/mocktests/"+test.ID.Hex()+"/submit", map[string]interface{}{
		"answers":   map[string]int{},
		"timeTaken": 60,
	}, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	ctx := context.Background()
	count, err := config.GetCollection("mocktestsattempts").CountDocuments(ctx,
		map[string]interface{}{"userId": userID, "mockTestId": test.ID})
	if err != nil {
		t.Fatalf("count error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 saved attempt in DB, got %d", count)
	}
}

// ─── GetAttemptDetails ────────────────────────────────────────────────────────

func TestGetAttemptDetails_InvalidID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("GET", "/attempts/:attemptId",
		setUserID(userID.Hex(), "9876543210"),
		GetAttemptDetails,
	)
	w := doRequest(r, "GET", "/attempts/bad-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestGetAttemptDetails_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")

	userID := primitive.NewObjectID()
	attemptID := primitive.NewObjectID()
	r := newRouter("GET", "/attempts/:attemptId",
		setUserID(userID.Hex(), "9876543210"),
		GetAttemptDetails,
	)
	w := doRequest(r, "GET", "/attempts/"+attemptID.Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestGetAttemptDetails_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")
	dropCollection(t, "questions")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	attempt := models.MockTestAttempt{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		MockTestID:  primitive.NewObjectID(),
		Answers:     []models.AttemptAnswer{},
		Score:       3,
		TotalMarks:  5,
		TimeTaken:   60,
		CompletedAt: time.Now(),
	}
	config.GetCollection("mocktestsattempts").InsertOne(ctx, attempt)

	r := newRouter("GET", "/attempts/:attemptId",
		setUserID(userID.Hex(), "9876543210"),
		GetAttemptDetails,
	)
	w := doRequest(r, "GET", "/attempts/"+attempt.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	result, ok := data["result"].(map[string]interface{})
	if !ok {
		t.Fatal("expected result in response")
	}
	if result["score"].(float64) != 3 {
		t.Errorf("score: want 3, got %v", result["score"])
	}
	if result["totalMarks"].(float64) != 5 {
		t.Errorf("totalMarks: want 5, got %v", result["totalMarks"])
	}
	if result["timeTaken"].(float64) != 60 {
		t.Errorf("timeTaken: want 60, got %v", result["timeTaken"])
	}
}

func TestGetAttemptDetails_ResultFieldsPresent(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	attempt := models.MockTestAttempt{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		MockTestID:  primitive.NewObjectID(),
		Answers:     []models.AttemptAnswer{},
		Score:       0,
		TotalMarks:  0,
		CompletedAt: time.Now(),
	}
	config.GetCollection("mocktestsattempts").InsertOne(ctx, attempt)

	r := newRouter("GET", "/attempts/:attemptId",
		setUserID(userID.Hex(), "9876543210"),
		GetAttemptDetails,
	)
	w := doRequest(r, "GET", "/attempts/"+attempt.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Fatalf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	result := data["result"].(map[string]interface{})

	requiredFields := []string{
		"attemptId", "score", "totalMarks", "correct", "wrong",
		"skipped", "percent", "timeTaken", "completedAt", "detailed",
	}
	for _, field := range requiredFields {
		if _, ok := result[field]; !ok {
			t.Errorf("missing field %q in attempt details result", field)
		}
	}
}

func TestGetAttemptDetails_WrongUser(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")

	ownerID := primitive.NewObjectID()
	otherID := primitive.NewObjectID()
	ctx := context.Background()
	attempt := models.MockTestAttempt{
		ID:          primitive.NewObjectID(),
		UserID:      ownerID, // belongs to ownerID
		MockTestID:  primitive.NewObjectID(),
		Score:       1,
		TotalMarks:  5,
		CompletedAt: time.Now(),
	}
	config.GetCollection("mocktestsattempts").InsertOne(ctx, attempt)

	// Request as otherID — filter {_id, userId} won't match → 404
	r := newRouter("GET", "/attempts/:attemptId",
		setUserID(otherID.Hex(), "9876543211"),
		GetAttemptDetails,
	)
	w := doRequest(r, "GET", "/attempts/"+attempt.ID.Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 when accessing another user's attempt, got %d", w.Code)
	}
}

// ─── GetUserAttempts ──────────────────────────────────────────────────────────

func TestGetUserAttempts_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/attempts", setUserID(userID.Hex(), "9876543210"), GetUserAttempts)
	w := doRequest(r, "GET", "/attempts", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	attempts, ok := data["attempts"].([]interface{})
	if !ok {
		t.Fatal("expected attempts array in response")
	}
	if len(attempts) != 0 {
		t.Errorf("expected 0 attempts, got %d", len(attempts))
	}
}

func TestGetUserAttempts_ReturnsOK(t *testing.T) {
	requireDB(t)

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/attempts", setUserID(userID.Hex(), "9876543210"), GetUserAttempts)
	w := doRequest(r, "GET", "/attempts", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestGetAttemptDetails_WithAnswersAndQuestions(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")
	dropCollection(t, "questions")

	userID := primitive.NewObjectID()
	ctx := context.Background()

	q1 := models.Question{
		ID: primitive.NewObjectID(), Text: "Q1", Subject: "Math",
		Options:      []models.QuestionOption{{Type: "text", Value: "A"}, {Type: "text", Value: "B"}},
		CorrectIndex: 0, CreatedAt: time.Now(),
	}
	q2 := models.Question{
		ID: primitive.NewObjectID(), Text: "Q2", Subject: "Math",
		Options:      []models.QuestionOption{{Type: "text", Value: "X"}, {Type: "text", Value: "Y"}},
		CorrectIndex: 1, CreatedAt: time.Now(),
	}
	config.GetCollection("questions").InsertOne(ctx, q1)
	config.GetCollection("questions").InsertOne(ctx, q2)

	attempt := models.MockTestAttempt{
		ID:     primitive.NewObjectID(),
		UserID: userID,
		Answers: []models.AttemptAnswer{
			{QuestionID: q1.ID, SelectedIndex: 0, IsCorrect: true},
			{QuestionID: q2.ID, SelectedIndex: 0, IsCorrect: false},
		},
		Score: 1, TotalMarks: 2, TimeTaken: 60, CompletedAt: time.Now(),
	}
	config.GetCollection("mocktestsattempts").InsertOne(ctx, attempt)

	r := newRouter("GET", "/attempts/:attemptId", setUserID(userID.Hex(), "9876543210"), GetAttemptDetails)
	w := doRequest(r, "GET", "/attempts/"+attempt.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	result := data["result"].(map[string]interface{})

	if result["correct"].(float64) != 1 {
		t.Errorf("correct: want 1, got %v", result["correct"])
	}
	if result["wrong"].(float64) != 1 {
		t.Errorf("wrong: want 1, got %v", result["wrong"])
	}
	if result["percent"].(float64) != 50 {
		t.Errorf("percent: want 50, got %v", result["percent"])
	}
	detailed, ok := result["detailed"].([]interface{})
	if !ok || len(detailed) != 2 {
		t.Errorf("expected 2 detailed results, got %v", result["detailed"])
	}
}

func TestGetAttemptDetails_SkippedAnswer(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")
	dropCollection(t, "questions")

	userID := primitive.NewObjectID()
	ctx := context.Background()

	q := models.Question{
		ID: primitive.NewObjectID(), Text: "Skipped Q", Subject: "Science",
		Options:      []models.QuestionOption{{Type: "text", Value: "A"}, {Type: "text", Value: "B"}},
		CorrectIndex: 0, CreatedAt: time.Now(),
	}
	config.GetCollection("questions").InsertOne(ctx, q)

	attempt := models.MockTestAttempt{
		ID: primitive.NewObjectID(), UserID: userID,
		Answers:     []models.AttemptAnswer{{QuestionID: q.ID, SelectedIndex: -1, IsCorrect: false}},
		Score:       0, TotalMarks: 1, CompletedAt: time.Now(),
	}
	config.GetCollection("mocktestsattempts").InsertOne(ctx, attempt)

	r := newRouter("GET", "/attempts/:attemptId", setUserID(userID.Hex(), "9876543210"), GetAttemptDetails)
	w := doRequest(r, "GET", "/attempts/"+attempt.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	result := parseBody(t, w)["data"].(map[string]interface{})["result"].(map[string]interface{})
	if result["skipped"].(float64) != 1 {
		t.Errorf("skipped: want 1, got %v", result["skipped"])
	}
}

func TestGetUserAttempts_OnlyOwnAttempts(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "mocktestsattempts")

	userID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()
	ctx := context.Background()

	// Create a real test so the $lookup doesn't drop the attempt
	test := insertMockTest(t, "Shared Test")

	// Own attempt
	config.GetCollection("mocktestsattempts").InsertOne(ctx, models.MockTestAttempt{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		MockTestID:  test.ID,
		Answers:     []models.AttemptAnswer{},
		Score:       2,
		TotalMarks:  5,
		CompletedAt: time.Now(),
	})
	// Another user's attempt — must not appear in own listing
	config.GetCollection("mocktestsattempts").InsertOne(ctx, models.MockTestAttempt{
		ID:          primitive.NewObjectID(),
		UserID:      otherUserID,
		MockTestID:  test.ID,
		Answers:     []models.AttemptAnswer{},
		Score:       4,
		TotalMarks:  5,
		CompletedAt: time.Now(),
	})

	r := newRouter("GET", "/attempts", setUserID(userID.Hex(), "9876543210"), GetUserAttempts)
	w := doRequest(r, "GET", "/attempts", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	attempts := data["attempts"].([]interface{})
	if len(attempts) != 1 {
		t.Errorf("expected 1 own attempt, got %d", len(attempts))
	}
}

func TestGetUserAttempts_MultipleAttemptsSorted(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktests")
	dropCollection(t, "mocktestsattempts")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	test := insertMockTest(t, "Multi Attempt Test")

	now := time.Now()
	for i := 0; i < 3; i++ {
		config.GetCollection("mocktestsattempts").InsertOne(ctx, models.MockTestAttempt{
			ID:          primitive.NewObjectID(),
			UserID:      userID,
			MockTestID:  test.ID,
			Answers:     []models.AttemptAnswer{},
			Score:       i,
			TotalMarks:  10,
			CompletedAt: now.Add(time.Duration(i) * time.Minute),
		})
	}

	r := newRouter("GET", "/attempts", setUserID(userID.Hex(), "9876543210"), GetUserAttempts)
	w := doRequest(r, "GET", "/attempts", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	attempts := data["attempts"].([]interface{})
	if len(attempts) != 3 {
		t.Errorf("expected 3 attempts, got %d", len(attempts))
	}
}
