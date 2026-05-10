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

func insertLiveClass(t *testing.T, isLive bool) models.LiveClass {
	t.Helper()
	class := models.LiveClass{
		ID:          primitive.NewObjectID(),
		Title:       "Test Class",
		Subject:     "Maths",
		TeacherName: "Prof. Test",
		ClassLevel:  "6",
		Duration:    60,
		IsLive:      isLive,
		CreatedAt:   time.Now(),
	}
	config.GetCollection("liveclasses").InsertOne(context.Background(), class)
	return class
}

// ─── CreateLiveClass ──────────────────────────────────────────────────────────

func TestCreateLiveClass_MissingFields(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/liveclasses",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		CreateLiveClass,
	)
	w := doRequest(r, "POST", "/admin/liveclasses", map[string]interface{}{
		"title": "Only Title",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing fields, got %d", w.Code)
	}
}

func TestCreateLiveClass_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "liveclasses")

	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/liveclasses",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		CreateLiveClass,
	)
	w := doRequest(r, "POST", "/admin/liveclasses", map[string]interface{}{
		"title":       "Live Maths",
		"subject":     "Maths",
		"teacherName": "Prof. Singh",
		"classLevel":  "7",
		"duration":    45,
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

// ─── ListAdminLiveClasses ─────────────────────────────────────────────────────

func TestListAdminLiveClasses_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "liveclasses")

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/liveclasses",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		ListAdminLiveClasses,
	)
	w := doRequest(r, "GET", "/admin/liveclasses", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	classes, ok := data["classes"].([]interface{})
	if !ok {
		t.Fatal("expected classes array")
	}
	if len(classes) != 0 {
		t.Errorf("expected 0 classes, got %d", len(classes))
	}
}

func TestListAdminLiveClasses_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "liveclasses")

	insertLiveClass(t, true)
	insertLiveClass(t, false)

	adminID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/liveclasses",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		ListAdminLiveClasses,
	)
	w := doRequest(r, "GET", "/admin/liveclasses", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	classes := data["classes"].([]interface{})
	if len(classes) != 2 {
		t.Errorf("expected 2 classes, got %d", len(classes))
	}
}

// ─── EndLiveClass ─────────────────────────────────────────────────────────────

func TestEndLiveClass_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/liveclasses/:id/end",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		EndLiveClass,
	)
	w := doRequest(r, "PUT", "/admin/liveclasses/bad-id/end", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestEndLiveClass_NonExistent_ReturnsOK(t *testing.T) {
	requireDB(t)
	dropCollection(t, "liveclasses")

	// UpdateOne does not error when no document matches — EndLiveClass returns 200
	adminID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/liveclasses/:id/end",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		EndLiveClass,
	)
	w := doRequest(r, "PUT", "/admin/liveclasses/"+classID.Hex()+"/end", nil, nil)
	if w.Code != http.StatusOK {
		t.Errorf("want 200 (UpdateOne with no match still succeeds), got %d", w.Code)
	}
}

func TestEndLiveClass_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "liveclasses")

	class := insertLiveClass(t, true)

	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/liveclasses/:id/end",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		EndLiveClass,
	)
	w := doRequest(r, "PUT", "/admin/liveclasses/"+class.ID.Hex()+"/end", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── ListActiveLiveClasses ────────────────────────────────────────────────────

func TestListActiveLiveClasses_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "liveclasses")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/liveclasses", setUserID(userID.Hex(), "9876543210"), ListActiveLiveClasses)
	w := doRequest(r, "GET", "/liveclasses", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	classes, ok := data["classes"].([]interface{})
	if !ok {
		t.Fatal("expected classes array")
	}
	if len(classes) != 0 {
		t.Errorf("expected 0 active classes, got %d", len(classes))
	}
}

func TestListActiveLiveClasses_OnlyLive(t *testing.T) {
	requireDB(t)
	dropCollection(t, "liveclasses")

	insertLiveClass(t, true)  // live
	insertLiveClass(t, false) // not live

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/liveclasses", setUserID(userID.Hex(), "9876543210"), ListActiveLiveClasses)
	w := doRequest(r, "GET", "/liveclasses", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	classes := data["classes"].([]interface{})
	if len(classes) != 1 {
		t.Errorf("expected 1 active class, got %d", len(classes))
	}
}

// ─── GetLiveClass ─────────────────────────────────────────────────────────────

func TestGetLiveClass_InvalidID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("GET", "/liveclasses/:id", setUserID(userID.Hex(), "9876543210"), GetLiveClass)
	w := doRequest(r, "GET", "/liveclasses/bad-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestGetLiveClass_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "liveclasses")

	userID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	r := newRouter("GET", "/liveclasses/:id", setUserID(userID.Hex(), "9876543210"), GetLiveClass)
	w := doRequest(r, "GET", "/liveclasses/"+classID.Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestGetLiveClass_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "liveclasses")

	class := insertLiveClass(t, true)

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/liveclasses/:id", setUserID(userID.Hex(), "9876543210"), GetLiveClass)
	w := doRequest(r, "GET", "/liveclasses/"+class.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── GetAgoraToken ────────────────────────────────────────────────────────────

func TestGetAgoraToken_InvalidID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("GET", "/liveclasses/:id/token", setUserID(userID.Hex(), "9876543210"), GetAgoraToken)
	w := doRequest(r, "GET", "/liveclasses/bad-id/token", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestGetAgoraToken_ValidID(t *testing.T) {
	userID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	r := newRouter("GET", "/liveclasses/:id/token", setUserID(userID.Hex(), "9876543210"), GetAgoraToken)
	w := doRequest(r, "GET", "/liveclasses/"+classID.Hex()+"/token", nil, nil)

	// Valid ID → returns token (may be empty if AGORA_APP_CERTIFICATE not set, but 200)
	if w.Code != http.StatusOK {
		t.Errorf("want 200 for valid class ID, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	if data["channelName"] != classID.Hex() {
		t.Errorf("channelName: want %s, got %v", classID.Hex(), data["channelName"])
	}
}

// ─── RegisterPushToken ────────────────────────────────────────────────────────

func TestRegisterPushToken_MissingToken(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/push-token", setUserID(userID.Hex(), "9876543210"), RegisterPushToken)
	w := doRequest(r, "POST", "/push-token", map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestRegisterPushToken_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "pushtokens")

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/push-token", setUserID(userID.Hex(), "9876543210"), RegisterPushToken)
	w := doRequest(r, "POST", "/push-token", map[string]interface{}{
		"token":    "fcm-token-12345",
		"platform": "android",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestRegisterPushToken_DefaultPlatform(t *testing.T) {
	requireDB(t)
	dropCollection(t, "pushtokens")

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/push-token", setUserID(userID.Hex(), "9876543210"), RegisterPushToken)
	w := doRequest(r, "POST", "/push-token", map[string]interface{}{
		"token": "apns-token-xyz",
		// no platform — defaults to "android"
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestRegisterPushToken_Upsert(t *testing.T) {
	requireDB(t)
	dropCollection(t, "pushtokens")

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/push-token", setUserID(userID.Hex(), "9876543210"), RegisterPushToken)

	// First call
	doRequest(r, "POST", "/push-token", map[string]interface{}{
		"token": "old-token",
	}, nil)

	// Second call — should upsert (not duplicate)
	w := doRequest(r, "POST", "/push-token", map[string]interface{}{
		"token": "new-token",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200 on upsert, got %d", w.Code)
	}
}

// ─── PushLiveQuestion ─────────────────────────────────────────────────────────

func TestPushLiveQuestion_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/live/classes/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		PushLiveQuestion,
	)
	w := doRequest(r, "POST", "/admin/live/classes/bad-id/questions", map[string]interface{}{
		"text": "Q1", "timerSeconds": 30,
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestPushLiveQuestion_MissingFields(t *testing.T) {
	adminID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/live/classes/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		PushLiveQuestion,
	)
	w := doRequest(r, "POST", "/admin/live/classes/"+classID.Hex()+"/questions", map[string]interface{}{
		"text": "Missing timerSeconds",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing timerSeconds, got %d", w.Code)
	}
}

func TestPushLiveQuestion_TooFewOptions(t *testing.T) {
	adminID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/live/classes/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		PushLiveQuestion,
	)
	w := doRequest(r, "POST", "/admin/live/classes/"+classID.Hex()+"/questions", map[string]interface{}{
		"text":         "Question?",
		"options":      []string{"Only one option"},
		"timerSeconds": 30,
		"isReadOnly":   false,
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for too few options, got %d", w.Code)
	}
}

func TestPushLiveQuestion_ReadOnlyNoOptions(t *testing.T) {
	requireDB(t)
	dropCollection(t, "livequestions")

	adminID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/live/classes/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		PushLiveQuestion,
	)
	w := doRequest(r, "POST", "/admin/live/classes/"+classID.Hex()+"/questions", map[string]interface{}{
		"text":         "Read-only announcement",
		"isReadOnly":   true,
		"timerSeconds": 30,
	}, nil)
	if w.Code != http.StatusCreated {
		t.Errorf("want 201 for read-only with no options, got %d: %s", w.Code, w.Body.String())
	}
}

func TestPushLiveQuestion_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "livequestions")

	adminID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/live/classes/:id/questions",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		PushLiveQuestion,
	)
	w := doRequest(r, "POST", "/admin/live/classes/"+classID.Hex()+"/questions", map[string]interface{}{
		"text":         "What is 2+2?",
		"options":      []string{"3", "4", "5", "6"},
		"correctIndex": 1,
		"timerSeconds": 30,
		"isReadOnly":   false,
	}, nil)
	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	question := data["question"].(map[string]interface{})
	if question["text"] != "What is 2+2?" {
		t.Errorf("text: want 'What is 2+2?', got %v", question["text"])
	}
}

// ─── EndLiveQuestion ──────────────────────────────────────────────────────────

func insertLiveQuestion(t *testing.T, classID primitive.ObjectID, isActive bool) models.LiveQuestion {
	t.Helper()
	q := models.LiveQuestion{
		ID:           primitive.NewObjectID(),
		LiveClassID:  classID,
		Text:         "Test Question",
		Options:      []string{"A", "B", "C", "D"},
		CorrectIndex: 0,
		IsReadOnly:   false,
		TimerSeconds: 30,
		IsActive:     isActive,
		CreatedAt:    time.Now(),
	}
	config.GetCollection("livequestions").InsertOne(context.Background(), q)
	return q
}

func TestEndLiveQuestion_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/live/classes/:id/questions/:qid",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		EndLiveQuestion,
	)
	w := doRequest(r, "DELETE", "/admin/live/classes/"+classID.Hex()+"/questions/bad-qid", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid question ID, got %d", w.Code)
	}
}

func TestEndLiveQuestion_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "livequestions")

	adminID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	questionID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/live/classes/:id/questions/:qid",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		EndLiveQuestion,
	)
	w := doRequest(r, "DELETE", "/admin/live/classes/"+classID.Hex()+"/questions/"+questionID.Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestEndLiveQuestion_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "livequestions")
	dropCollection(t, "quizanswers")

	classID := primitive.NewObjectID()
	q := insertLiveQuestion(t, classID, true)

	adminID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/live/classes/:id/questions/:qid",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		EndLiveQuestion,
	)
	w := doRequest(r, "DELETE", "/admin/live/classes/"+classID.Hex()+"/questions/"+q.ID.Hex(), nil, nil)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── GetQuestionLeaderboard ───────────────────────────────────────────────────

func TestGetQuestionLeaderboard_InvalidID(t *testing.T) {
	adminID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/live/classes/:id/questions/:qid/leaderboard",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		GetQuestionLeaderboard,
	)
	w := doRequest(r, "GET", "/admin/live/classes/"+classID.Hex()+"/questions/bad-qid/leaderboard", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestGetQuestionLeaderboard_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "quizanswers")

	adminID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	questionID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/live/classes/:id/questions/:qid/leaderboard",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		GetQuestionLeaderboard,
	)
	w := doRequest(r, "GET", "/admin/live/classes/"+classID.Hex()+"/questions/"+questionID.Hex()+"/leaderboard", nil, nil)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	if _, ok := data["leaderboard"]; !ok {
		t.Error("expected leaderboard field in response")
	}
}
