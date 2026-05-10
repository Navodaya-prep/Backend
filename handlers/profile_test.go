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

// ─── GetProfile ───────────────────────────────────────────────────────────────

func TestGetProfile_InvalidID(t *testing.T) {
	r := newRouter("GET", "/profile", setUserID("not-valid-hex", "9876543210"), GetProfile)
	w := doRequest(r, "GET", "/profile", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid user ID, got %d", w.Code)
	}
}

func TestGetProfile_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "users")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/profile", setUserID(userID.Hex(), "9876543210"), GetProfile)
	w := doRequest(r, "GET", "/profile", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for missing user, got %d", w.Code)
	}
}

func TestGetProfile_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "users")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	config.GetCollection("users").InsertOne(ctx, models.User{
		ID:        userID,
		Name:      "Test Student",
		Phone:     "9876543210",
		ClassLevel: "10",
		State:     "Delhi",
		Streak:    3,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	r := newRouter("GET", "/profile", setUserID(userID.Hex(), "9876543210"), GetProfile)
	w := doRequest(r, "GET", "/profile", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	user := data["user"].(map[string]interface{})
	if user["name"] != "Test Student" {
		t.Errorf("name: want Test Student, got %v", user["name"])
	}
	// stats should be present
	if data["stats"] == nil {
		t.Error("expected stats in response")
	}
}

func TestGetProfile_StatsDefaults(t *testing.T) {
	requireDB(t)
	dropCollection(t, "users")
	dropCollection(t, "mocktestsattempts")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	config.GetCollection("users").InsertOne(ctx, models.User{
		ID:    userID,
		Name:  "No Tests",
		Phone: "9876543211",
	})

	r := newRouter("GET", "/profile", setUserID(userID.Hex(), "9876543211"), GetProfile)
	w := doRequest(r, "GET", "/profile", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	stats := data["stats"].(map[string]interface{})
	// With no test attempts, all stats should be 0
	if stats["totalTests"].(float64) != 0 {
		t.Errorf("totalTests: want 0, got %v", stats["totalTests"])
	}
}

// ─── UpdateProfile ────────────────────────────────────────────────────────────

func TestUpdateProfile_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "users")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	config.GetCollection("users").InsertOne(ctx, models.User{
		ID:    userID,
		Name:  "Old Name",
		State: "Delhi",
		Phone: "9876543210",
	})

	r := newRouter("PUT", "/profile", setUserID(userID.Hex(), "9876543210"), UpdateProfile)
	w := doRequest(r, "PUT", "/profile", map[string]interface{}{
		"name":  "New Name",
		"state": "Mumbai",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

func TestUpdateProfile_PartialUpdate_NameOnly(t *testing.T) {
	requireDB(t)
	dropCollection(t, "users")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	config.GetCollection("users").InsertOne(ctx, models.User{
		ID:    userID,
		Name:  "Old Name",
		State: "Delhi",
		Phone: "9876543210",
	})

	r := newRouter("PUT", "/profile", setUserID(userID.Hex(), "9876543210"), UpdateProfile)
	w := doRequest(r, "PUT", "/profile", map[string]interface{}{
		"name": "Updated Name",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestUpdateProfile_EmptyBody(t *testing.T) {
	requireDB(t)
	dropCollection(t, "users")

	userID := primitive.NewObjectID()
	ctx := context.Background()
	config.GetCollection("users").InsertOne(ctx, models.User{
		ID:    userID,
		Name:  "Name",
		Phone: "9876543210",
	})

	r := newRouter("PUT", "/profile", setUserID(userID.Hex(), "9876543210"), UpdateProfile)
	w := doRequest(r, "PUT", "/profile", map[string]interface{}{}, nil)

	// Empty body is valid — no-op update
	if w.Code != http.StatusOK {
		t.Errorf("want 200 for empty body (no-op), got %d", w.Code)
	}
}
