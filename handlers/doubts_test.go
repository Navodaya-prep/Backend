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

// ─── ListDoubts ───────────────────────────────────────────────────────────────

func TestListDoubts_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	r := newRouter("GET", "/doubts", ListDoubts)
	w := doRequest(r, "GET", "/doubts", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	doubts, ok := data["doubts"].([]interface{})
	if !ok {
		t.Fatal("expected doubts array in response")
	}
	if len(doubts) != 0 {
		t.Errorf("expected 0 doubts on empty collection, got %d", len(doubts))
	}
}

func TestListDoubts_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		config.GetCollection("doubts").InsertOne(ctx, models.Doubt{
			ID:        primitive.NewObjectID(),
			UserID:    userID,
			UserName:  "Student",
			Subject:   "Math",
			Text:      "Question text",
			Status:    "open",
			CreatedAt: time.Now(),
		})
	}

	r := newRouter("GET", "/doubts", ListDoubts)
	w := doRequest(r, "GET", "/doubts", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	doubts := data["doubts"].([]interface{})
	if len(doubts) != 3 {
		t.Errorf("expected 3 doubts, got %d", len(doubts))
	}
}

func TestListDoubts_FilterBySubject_Match(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	config.GetCollection("doubts").InsertOne(ctx, models.Doubt{
		ID: primitive.NewObjectID(), UserID: userID, UserName: "S",
		Subject: "Physics", Text: "Q1", Status: "open", CreatedAt: time.Now(),
	})
	config.GetCollection("doubts").InsertOne(ctx, models.Doubt{
		ID: primitive.NewObjectID(), UserID: userID, UserName: "S",
		Subject: "Chemistry", Text: "Q2", Status: "open", CreatedAt: time.Now(),
	})

	r := newRouter("GET", "/doubts", ListDoubts)
	w := doRequest(r, "GET", "/doubts?subject=Physics", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	doubts := data["doubts"].([]interface{})
	if len(doubts) != 1 {
		t.Errorf("expected 1 Physics doubt, got %d", len(doubts))
	}
}

func TestListDoubts_FilterBySubject_NoMatch(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	config.GetCollection("doubts").InsertOne(ctx, models.Doubt{
		ID: primitive.NewObjectID(), UserID: userID, UserName: "S",
		Subject: "Biology", Text: "Q", Status: "open", CreatedAt: time.Now(),
	})

	r := newRouter("GET", "/doubts", ListDoubts)
	w := doRequest(r, "GET", "/doubts?subject=Math", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	doubts := data["doubts"].([]interface{})
	if len(doubts) != 0 {
		t.Errorf("expected 0 doubts for non-matching subject, got %d", len(doubts))
	}
}

func TestListDoubts_AnswerCountAttached(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID: primitive.NewObjectID(), UserID: userID, UserName: "S",
		Subject: "Math", Text: "Q", Status: "open", CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)
	// Insert 2 answers for the doubt
	for i := 0; i < 2; i++ {
		config.GetCollection("doubtanswers").InsertOne(ctx, models.DoubtAnswer{
			ID:      primitive.NewObjectID(),
			DoubtID: doubt.ID,
			UserID:  primitive.NewObjectID(),
			Text:    "Answer",
		})
	}

	r := newRouter("GET", "/doubts", ListDoubts)
	w := doRequest(r, "GET", "/doubts", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	doubts := data["doubts"].([]interface{})
	if len(doubts) == 0 {
		t.Fatal("expected at least 1 doubt")
	}
	d := doubts[0].(map[string]interface{})
	if d["answerCount"].(float64) != 2 {
		t.Errorf("expected answerCount=2, got %v", d["answerCount"])
	}
}

// ─── PostDoubt ────────────────────────────────────────────────────────────────

func TestPostDoubt_NilBody(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/doubts", setUserID(userID.Hex(), "9876543210"), PostDoubt)
	// nil body → EOF → 400
	w := doRequest(r, "POST", "/doubts", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for nil body, got %d", w.Code)
	}
}

func TestPostDoubt_MissingSubject(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/doubts", setUserID(userID.Hex(), "9876543210"), PostDoubt)
	w := doRequest(r, "POST", "/doubts", map[string]interface{}{
		"text": "My question",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing subject, got %d", w.Code)
	}
}

func TestPostDoubt_MissingText(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/doubts", setUserID(userID.Hex(), "9876543210"), PostDoubt)
	w := doRequest(r, "POST", "/doubts", map[string]interface{}{
		"subject": "Math",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing text, got %d", w.Code)
	}
}

func TestPostDoubt_MissingBothFields(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/doubts", setUserID(userID.Hex(), "9876543210"), PostDoubt)
	w := doRequest(r, "POST", "/doubts", map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty body, got %d", w.Code)
	}
}

func TestPostDoubt_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/doubts", setUserID(userID.Hex(), "9876543210"), PostDoubt)
	w := doRequest(r, "POST", "/doubts", map[string]interface{}{
		"subject": "Physics",
		"text":    "What is Newton's first law?",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	doubt, ok := data["doubt"].(map[string]interface{})
	if !ok {
		t.Fatal("expected doubt object in response")
	}
	if doubt["subject"] != "Physics" {
		t.Errorf("subject: want Physics, got %v", doubt["subject"])
	}
	if doubt["text"] != "What is Newton's first law?" {
		t.Errorf("text mismatch, got %v", doubt["text"])
	}
	if doubt["status"] != "open" {
		t.Errorf("status: want open, got %v", doubt["status"])
	}
}

func TestPostDoubt_WithOptionalImageURL(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/doubts", setUserID(userID.Hex(), "9876543210"), PostDoubt)
	w := doRequest(r, "POST", "/doubts", map[string]interface{}{
		"subject":  "Chemistry",
		"text":     "What is H2O?",
		"imageUrl": "https://example.com/image.png",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	doubt := data["doubt"].(map[string]interface{})
	if doubt["imageUrl"] != "https://example.com/image.png" {
		t.Errorf("imageUrl: want set value, got %v", doubt["imageUrl"])
	}
}

// ─── UpdateDoubt ──────────────────────────────────────────────────────────────

func TestUpdateDoubt_InvalidID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("PUT", "/doubts/:id", setUserID(userID.Hex(), "9876543210"), UpdateDoubt)
	w := doRequest(r, "PUT", "/doubts/not-a-valid-id", map[string]interface{}{
		"subject": "Math", "text": "Updated",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestUpdateDoubt_MissingFields(t *testing.T) {
	userID := primitive.NewObjectID()
	doubtID := primitive.NewObjectID()
	r := newRouter("PUT", "/doubts/:id", setUserID(userID.Hex(), "9876543210"), UpdateDoubt)
	w := doRequest(r, "PUT", "/doubts/"+doubtID.Hex(), map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing fields, got %d", w.Code)
	}
}

func TestUpdateDoubt_NilBody(t *testing.T) {
	userID := primitive.NewObjectID()
	doubtID := primitive.NewObjectID()
	r := newRouter("PUT", "/doubts/:id", setUserID(userID.Hex(), "9876543210"), UpdateDoubt)
	w := doRequest(r, "PUT", "/doubts/"+doubtID.Hex(), nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for nil body, got %d", w.Code)
	}
}

func TestUpdateDoubt_NotOwner(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")

	ownerID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    ownerID,
		UserName:  "Owner",
		Subject:   "Math",
		Text:      "Original",
		Status:    "open",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)

	// Use a different user ID — should not match the filter
	r := newRouter("PUT", "/doubts/:id", setUserID(otherUserID.Hex(), "9876543211"), UpdateDoubt)
	w := doRequest(r, "PUT", "/doubts/"+doubt.ID.Hex(), map[string]interface{}{
		"subject": "Math",
		"text":    "Attempted hijack",
	}, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 when not owner, got %d", w.Code)
	}
}

func TestUpdateDoubt_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		UserName:  "Student",
		Subject:   "Math",
		Text:      "Old text",
		Status:    "open",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)

	r := newRouter("PUT", "/doubts/:id", setUserID(userID.Hex(), "9876543210"), UpdateDoubt)
	w := doRequest(r, "PUT", "/doubts/"+doubt.ID.Hex(), map[string]interface{}{
		"subject": "Physics",
		"text":    "Updated text",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

// ─── GetDoubtAnswers ──────────────────────────────────────────────────────────

func TestGetDoubtAnswers_InvalidID(t *testing.T) {
	r := newRouter("GET", "/doubts/:id/answers", GetDoubtAnswers)
	w := doRequest(r, "GET", "/doubts/invalid-id/answers", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestGetDoubtAnswers_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")

	r := newRouter("GET", "/doubts/:id/answers", GetDoubtAnswers)
	w := doRequest(r, "GET", "/doubts/"+primitive.NewObjectID().Hex()+"/answers", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for missing doubt, got %d", w.Code)
	}
}

func TestGetDoubtAnswers_EmptyAnswers(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		UserName:  "S",
		Subject:   "Math",
		Text:      "Q",
		Status:    "open",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)

	r := newRouter("GET", "/doubts/:id/answers", GetDoubtAnswers)
	w := doRequest(r, "GET", "/doubts/"+doubt.ID.Hex()+"/answers", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	answers, ok := data["answers"].([]interface{})
	if !ok {
		t.Fatal("expected answers array in response")
	}
	if len(answers) != 0 {
		t.Errorf("expected 0 answers, got %d", len(answers))
	}
	if data["doubt"] == nil {
		t.Error("expected doubt in response")
	}
}

func TestGetDoubtAnswers_WithAnswers(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		UserName:  "S",
		Subject:   "Science",
		Text:      "How does photosynthesis work?",
		Status:    "open",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)

	for i := 0; i < 2; i++ {
		config.GetCollection("doubtanswers").InsertOne(ctx, models.DoubtAnswer{
			ID:        primitive.NewObjectID(),
			DoubtID:   doubt.ID,
			UserID:    primitive.NewObjectID(),
			UserName:  "Answerer",
			IsAdmin:   false,
			Text:      "Answer text",
			CreatedAt: time.Now(),
		})
	}

	r := newRouter("GET", "/doubts/:id/answers", GetDoubtAnswers)
	w := doRequest(r, "GET", "/doubts/"+doubt.ID.Hex()+"/answers", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	answers := data["answers"].([]interface{})
	if len(answers) != 2 {
		t.Errorf("expected 2 answers, got %d", len(answers))
	}
}

// ─── PostDoubtAnswer ──────────────────────────────────────────────────────────

func TestPostDoubtAnswer_InvalidID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/doubts/:id/answers",
		setUserID(userID.Hex(), "9876543210"),
		PostDoubtAnswer,
	)
	w := doRequest(r, "POST", "/doubts/bad-id/answers", map[string]interface{}{
		"text": "My answer",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid doubt ID, got %d", w.Code)
	}
}

func TestPostDoubtAnswer_MissingText(t *testing.T) {
	userID := primitive.NewObjectID()
	doubtID := primitive.NewObjectID()
	r := newRouter("POST", "/doubts/:id/answers",
		setUserID(userID.Hex(), "9876543210"),
		PostDoubtAnswer,
	)
	w := doRequest(r, "POST", "/doubts/"+doubtID.Hex()+"/answers", map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing text, got %d", w.Code)
	}
}

func TestPostDoubtAnswer_NilBody(t *testing.T) {
	userID := primitive.NewObjectID()
	doubtID := primitive.NewObjectID()
	r := newRouter("POST", "/doubts/:id/answers",
		setUserID(userID.Hex(), "9876543210"),
		PostDoubtAnswer,
	)
	w := doRequest(r, "POST", "/doubts/"+doubtID.Hex()+"/answers", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for nil body, got %d", w.Code)
	}
}

func TestPostDoubtAnswer_DoubtNotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/doubts/:id/answers",
		setUserID(userID.Hex(), "9876543210"),
		PostDoubtAnswer,
	)
	w := doRequest(r, "POST", "/doubts/"+primitive.NewObjectID().Hex()+"/answers",
		map[string]interface{}{"text": "Answer"}, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 when doubt not found, got %d", w.Code)
	}
}

func TestPostDoubtAnswer_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    primitive.NewObjectID(),
		UserName:  "Asker",
		Subject:   "Math",
		Text:      "Q",
		Status:    "open",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)

	r := newRouter("POST", "/doubts/:id/answers",
		setUserID(userID.Hex(), "9876543210"),
		PostDoubtAnswer,
	)
	w := doRequest(r, "POST", "/doubts/"+doubt.ID.Hex()+"/answers",
		map[string]interface{}{"text": "Here is your answer"}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	answer, ok := data["answer"].(map[string]interface{})
	if !ok {
		t.Fatal("expected answer in response")
	}
	if answer["text"] != "Here is your answer" {
		t.Errorf("answer text mismatch, got %v", answer["text"])
	}
	if answer["isAdmin"] != false {
		t.Error("expected isAdmin=false for student answer")
	}
}

// ─── DeleteDoubt ──────────────────────────────────────────────────────────────

func TestDeleteDoubt_InvalidID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("DELETE", "/doubts/:id", setUserID(userID.Hex(), "9876543210"), DeleteDoubt)
	w := doRequest(r, "DELETE", "/doubts/bad-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestDeleteDoubt_NotOwner(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")

	ownerID := primitive.NewObjectID()
	otherUserID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    ownerID,
		UserName:  "Owner",
		Subject:   "Math",
		Text:      "Q",
		Status:    "open",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)

	// Filter is {_id: doubtID, userId: userID} — different user means 404
	r := newRouter("DELETE", "/doubts/:id", setUserID(otherUserID.Hex(), "9876543211"), DeleteDoubt)
	w := doRequest(r, "DELETE", "/doubts/"+doubt.ID.Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 when deleting another user's doubt, got %d", w.Code)
	}
}

func TestDeleteDoubt_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")

	userID := primitive.NewObjectID()
	r := newRouter("DELETE", "/doubts/:id", setUserID(userID.Hex(), "9876543210"), DeleteDoubt)
	w := doRequest(r, "DELETE", "/doubts/"+primitive.NewObjectID().Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent doubt, got %d", w.Code)
	}
}

func TestDeleteDoubt_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		UserName:  "Student",
		Subject:   "Math",
		Text:      "Delete me",
		Status:    "open",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)

	r := newRouter("DELETE", "/doubts/:id", setUserID(userID.Hex(), "9876543210"), DeleteDoubt)
	w := doRequest(r, "DELETE", "/doubts/"+doubt.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

func TestDeleteDoubt_AlsoCleansAnswers(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    userID,
		UserName:  "Student",
		Subject:   "Math",
		Text:      "With answers",
		Status:    "answered",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)
	config.GetCollection("doubtanswers").InsertOne(ctx, models.DoubtAnswer{
		ID:      primitive.NewObjectID(),
		DoubtID: doubt.ID,
		UserID:  primitive.NewObjectID(),
		Text:    "An answer",
	})

	r := newRouter("DELETE", "/doubts/:id", setUserID(userID.Hex(), "9876543210"), DeleteDoubt)
	w := doRequest(r, "DELETE", "/doubts/"+doubt.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	// Verify answers were also removed
	count, _ := config.GetCollection("doubtanswers").CountDocuments(ctx, map[string]interface{}{
		"doubtId": doubt.ID,
	})
	if count != 0 {
		t.Errorf("expected associated answers to be deleted, got count=%d", count)
	}
}

// ─── AdminListDoubts ──────────────────────────────────────────────────────────

func TestAdminListDoubts_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/doubts",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListDoubts,
	)
	w := doRequest(r, "GET", "/admin/doubts", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	doubts := data["doubts"].([]interface{})
	if len(doubts) != 0 {
		t.Errorf("expected 0 doubts, got %d", len(doubts))
	}
}

func TestAdminListDoubts_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	ctx := context.Background()
	for i := 0; i < 2; i++ {
		config.GetCollection("doubts").InsertOne(ctx, models.Doubt{
			ID:        primitive.NewObjectID(),
			UserID:    primitive.NewObjectID(),
			UserName:  "Student",
			Subject:   "Biology",
			Text:      "Question",
			Status:    "open",
			CreatedAt: time.Now(),
		})
	}

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/doubts",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListDoubts,
	)
	w := doRequest(r, "GET", "/admin/doubts", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	doubts := data["doubts"].([]interface{})
	if len(doubts) != 2 {
		t.Errorf("expected 2 doubts, got %d", len(doubts))
	}
}

func TestAdminListDoubts_FilterByStatus(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	ctx := context.Background()
	config.GetCollection("doubts").InsertOne(ctx, models.Doubt{
		ID: primitive.NewObjectID(), UserID: primitive.NewObjectID(),
		Subject: "Math", Text: "Open Q", Status: "open", CreatedAt: time.Now(),
	})
	config.GetCollection("doubts").InsertOne(ctx, models.Doubt{
		ID: primitive.NewObjectID(), UserID: primitive.NewObjectID(),
		Subject: "Math", Text: "Answered Q", Status: "answered", CreatedAt: time.Now(),
	})

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/doubts",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminListDoubts,
	)
	w := doRequest(r, "GET", "/admin/doubts?status=open", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	doubts := data["doubts"].([]interface{})
	if len(doubts) != 1 {
		t.Errorf("expected 1 open doubt, got %d", len(doubts))
	}
}

// ─── AdminAnswerDoubt ─────────────────────────────────────────────────────────

func TestAdminAnswerDoubt_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/doubts/:id/answer",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminAnswerDoubt,
	)
	w := doRequest(r, "POST", "/admin/doubts/bad-id/answer",
		map[string]interface{}{"text": "Answer"}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestAdminAnswerDoubt_MissingText(t *testing.T) {
	adminID := primitive.NewObjectID()
	doubtID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/doubts/:id/answer",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminAnswerDoubt,
	)
	w := doRequest(r, "POST", "/admin/doubts/"+doubtID.Hex()+"/answer",
		map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing text, got %d", w.Code)
	}
}

func TestAdminAnswerDoubt_NilBody(t *testing.T) {
	adminID := primitive.NewObjectID()
	doubtID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/doubts/:id/answer",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminAnswerDoubt,
	)
	w := doRequest(r, "POST", "/admin/doubts/"+doubtID.Hex()+"/answer", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for nil body, got %d", w.Code)
	}
}

func TestAdminAnswerDoubt_DoubtNotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")

	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/doubts/:id/answer",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminAnswerDoubt,
	)
	w := doRequest(r, "POST", "/admin/doubts/"+primitive.NewObjectID().Hex()+"/answer",
		map[string]interface{}{"text": "Admin answer"}, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 when doubt not found, got %d", w.Code)
	}
}

func TestAdminAnswerDoubt_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	adminID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    primitive.NewObjectID(),
		UserName:  "Student",
		Subject:   "History",
		Text:      "When did WWII end?",
		Status:    "open",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)

	r := newRouter("POST", "/admin/doubts/:id/answer",
		setAdminID(adminID.Hex(), "admin@school.com", true),
		AdminAnswerDoubt,
	)
	w := doRequest(r, "POST", "/admin/doubts/"+doubt.ID.Hex()+"/answer",
		map[string]interface{}{"text": "WWII ended in 1945"}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	answer, ok := data["answer"].(map[string]interface{})
	if !ok {
		t.Fatal("expected answer in response")
	}
	if answer["isAdmin"] != true {
		t.Error("expected isAdmin=true for admin answer")
	}
	if answer["text"] != "WWII ended in 1945" {
		t.Errorf("answer text mismatch, got %v", answer["text"])
	}
}

// ─── AdminGetDoubtAnswers ─────────────────────────────────────────────────────

func TestAdminGetDoubtAnswers_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/doubts/:id/answers",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminGetDoubtAnswers,
	)
	w := doRequest(r, "GET", "/admin/doubts/not-valid/answers", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestAdminGetDoubtAnswers_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/doubts/:id/answers",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminGetDoubtAnswers,
	)
	w := doRequest(r, "GET", "/admin/doubts/"+primitive.NewObjectID().Hex()+"/answers", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for missing doubt, got %d", w.Code)
	}
}

func TestAdminGetDoubtAnswers_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	adminID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    primitive.NewObjectID(),
		UserName:  "S",
		Subject:   "Math",
		Text:      "Q",
		Status:    "open",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)
	config.GetCollection("doubtanswers").InsertOne(ctx, models.DoubtAnswer{
		ID:        primitive.NewObjectID(),
		DoubtID:   doubt.ID,
		UserID:    adminID,
		UserName:  "admin@test.com",
		IsAdmin:   true,
		Text:      "Admin reply",
		CreatedAt: time.Now(),
	})

	r := newRouter("GET", "/admin/doubts/:id/answers",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminGetDoubtAnswers,
	)
	w := doRequest(r, "GET", "/admin/doubts/"+doubt.ID.Hex()+"/answers", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	if data["doubt"] == nil {
		t.Error("expected doubt in response")
	}
	answers := data["answers"].([]interface{})
	if len(answers) != 1 {
		t.Errorf("expected 1 answer, got %d", len(answers))
	}
}

// ─── AdminDeleteDoubt ─────────────────────────────────────────────────────────

func TestAdminDeleteDoubt_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/doubts/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteDoubt,
	)
	w := doRequest(r, "DELETE", "/admin/doubts/bad-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestAdminDeleteDoubt_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/doubts/:id",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		AdminDeleteDoubt,
	)
	w := doRequest(r, "DELETE", "/admin/doubts/"+primitive.NewObjectID().Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for missing doubt, got %d", w.Code)
	}
}

func TestAdminDeleteDoubt_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	adminID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    primitive.NewObjectID(),
		UserName:  "Student",
		Subject:   "Math",
		Text:      "Admin will delete this",
		Status:    "open",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)

	r := newRouter("DELETE", "/admin/doubts/:id",
		setAdminID(adminID.Hex(), "admin@test.com", true),
		AdminDeleteDoubt,
	)
	w := doRequest(r, "DELETE", "/admin/doubts/"+doubt.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

func TestAdminDeleteDoubt_AlsoCleansAnswers(t *testing.T) {
	requireDB(t)
	dropCollection(t, "doubts")
	dropCollection(t, "doubtanswers")

	adminID := primitive.NewObjectID()
	ctx := context.Background()
	doubt := models.Doubt{
		ID:        primitive.NewObjectID(),
		UserID:    primitive.NewObjectID(),
		UserName:  "Student",
		Subject:   "Bio",
		Text:      "With answer",
		Status:    "answered",
		CreatedAt: time.Now(),
	}
	config.GetCollection("doubts").InsertOne(ctx, doubt)
	config.GetCollection("doubtanswers").InsertOne(ctx, models.DoubtAnswer{
		ID: primitive.NewObjectID(), DoubtID: doubt.ID,
		UserID: adminID, Text: "Admin answer",
	})

	r := newRouter("DELETE", "/admin/doubts/:id",
		setAdminID(adminID.Hex(), "admin@test.com", true),
		AdminDeleteDoubt,
	)
	w := doRequest(r, "DELETE", "/admin/doubts/"+doubt.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	count, _ := config.GetCollection("doubtanswers").CountDocuments(ctx, map[string]interface{}{
		"doubtId": doubt.ID,
	})
	if count != 0 {
		t.Errorf("expected associated answers deleted, got count=%d", count)
	}
}
