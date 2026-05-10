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

// ─── AddBookmark ──────────────────────────────────────────────────────────────

func TestAddBookmark_MissingQuestionID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/bookmarks", setUserID(userID.Hex(), "9876543210"), AddBookmark)
	w := doRequest(r, "POST", "/bookmarks", map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing questionId, got %d", w.Code)
	}
}

func TestAddBookmark_InvalidQuestionID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/bookmarks", setUserID(userID.Hex(), "9876543210"), AddBookmark)
	w := doRequest(r, "POST", "/bookmarks", map[string]interface{}{
		"questionId": "not-a-valid-id",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid questionId, got %d", w.Code)
	}
}

func TestAddBookmark_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "bookmarks")

	userID := primitive.NewObjectID()
	questionID := primitive.NewObjectID()

	r := newRouter("POST", "/bookmarks", setUserID(userID.Hex(), "9876543210"), AddBookmark)
	w := doRequest(r, "POST", "/bookmarks", map[string]interface{}{
		"questionId": questionID.Hex(),
		"source":     "practice",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

func TestAddBookmark_Idempotent(t *testing.T) {
	requireDB(t)
	dropCollection(t, "bookmarks")

	userID := primitive.NewObjectID()
	questionID := primitive.NewObjectID()

	r := newRouter("POST", "/bookmarks", setUserID(userID.Hex(), "9876543210"), AddBookmark)
	body := map[string]interface{}{
		"questionId": questionID.Hex(),
	}

	// First call creates
	w1 := doRequest(r, "POST", "/bookmarks", body, nil)
	if w1.Code != http.StatusCreated {
		t.Errorf("first call: want 201, got %d", w1.Code)
	}

	// Second call is idempotent — 200 "Already bookmarked"
	w2 := doRequest(r, "POST", "/bookmarks", body, nil)
	if w2.Code != http.StatusOK {
		t.Errorf("second call: want 200 (already bookmarked), got %d", w2.Code)
	}
}

func TestAddBookmark_DefaultSource(t *testing.T) {
	requireDB(t)
	dropCollection(t, "bookmarks")

	userID := primitive.NewObjectID()
	questionID := primitive.NewObjectID()

	r := newRouter("POST", "/bookmarks", setUserID(userID.Hex(), "9876543210"), AddBookmark)
	w := doRequest(r, "POST", "/bookmarks", map[string]interface{}{
		"questionId": questionID.Hex(),
		// no source — should default to "practice"
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", w.Code)
	}
}

// ─── RemoveBookmark ───────────────────────────────────────────────────────────

func TestRemoveBookmark_InvalidID(t *testing.T) {
	userID := primitive.NewObjectID()
	r := newRouter("DELETE", "/bookmarks/:questionId",
		setUserID(userID.Hex(), "9876543210"),
		RemoveBookmark,
	)
	w := doRequest(r, "DELETE", "/bookmarks/invalid-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid question ID, got %d", w.Code)
	}
}

func TestRemoveBookmark_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "bookmarks")

	userID := primitive.NewObjectID()
	questionID := primitive.NewObjectID()

	r := newRouter("DELETE", "/bookmarks/:questionId",
		setUserID(userID.Hex(), "9876543210"),
		RemoveBookmark,
	)
	w := doRequest(r, "DELETE", "/bookmarks/"+questionID.Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent bookmark, got %d", w.Code)
	}
}

func TestRemoveBookmark_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "bookmarks")

	userID := primitive.NewObjectID()
	questionID := primitive.NewObjectID()
	ctx := context.Background()
	config.GetCollection("bookmarks").InsertOne(ctx, models.Bookmark{
		ID:         primitive.NewObjectID(),
		UserID:     userID,
		QuestionID: questionID,
		Source:     "practice",
		CreatedAt:  time.Now(),
	})

	r := newRouter("DELETE", "/bookmarks/:questionId",
		setUserID(userID.Hex(), "9876543210"),
		RemoveBookmark,
	)
	w := doRequest(r, "DELETE", "/bookmarks/"+questionID.Hex(), nil, nil)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── ListBookmarks ────────────────────────────────────────────────────────────

func TestListBookmarks_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "bookmarks")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/bookmarks", setUserID(userID.Hex(), "9876543210"), ListBookmarks)
	w := doRequest(r, "GET", "/bookmarks", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	bms, ok := data["bookmarks"].([]interface{})
	if !ok {
		t.Fatal("expected bookmarks array")
	}
	if len(bms) != 0 {
		t.Errorf("expected 0 bookmarks, got %d", len(bms))
	}
}

func TestListBookmarks_WithEntries(t *testing.T) {
	requireDB(t)
	dropCollection(t, "bookmarks")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	for i := 0; i < 3; i++ {
		config.GetCollection("bookmarks").InsertOne(ctx, models.Bookmark{
			ID:         primitive.NewObjectID(),
			UserID:     userID,
			QuestionID: primitive.NewObjectID(),
			Source:     "practice",
			CreatedAt:  time.Now(),
		})
	}

	r := newRouter("GET", "/bookmarks", setUserID(userID.Hex(), "9876543210"), ListBookmarks)
	w := doRequest(r, "GET", "/bookmarks", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	bms := data["bookmarks"].([]interface{})
	if len(bms) != 3 {
		t.Errorf("expected 3 bookmarks, got %d", len(bms))
	}
}
