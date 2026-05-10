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

// ─── GetSettings ──────────────────────────────────────────────────────────────

func TestGetSettings_NoRecord_ReturnsDefaults(t *testing.T) {
	requireDB(t)
	dropCollection(t, "settings")

	r := newRouter("GET", "/settings", GetSettings)
	w := doRequest(r, "GET", "/settings", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	settings := data["settings"].(map[string]interface{})
	// Default ExamName should be set
	if settings["examName"] == nil || settings["examName"] == "" {
		t.Error("expected default examName in settings")
	}
}

func TestGetSettings_WithExistingRecord(t *testing.T) {
	requireDB(t)
	dropCollection(t, "settings")

	ctx := context.Background()
	config.GetCollection("settings").InsertOne(ctx, models.Settings{
		ID:       primitive.NewObjectID(),
		ExamDate: time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		ExamName: "JNVST 2026",
	})

	r := newRouter("GET", "/settings", GetSettings)
	w := doRequest(r, "GET", "/settings", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	settings := data["settings"].(map[string]interface{})
	if settings["examName"] != "JNVST 2026" {
		t.Errorf("examName: want JNVST 2026, got %v", settings["examName"])
	}
}

func TestGetSettings_PublicEndpoint_NoAuthRequired(t *testing.T) {
	requireDB(t)
	dropCollection(t, "settings")
	// GetSettings is a public endpoint — no auth middleware needed
	r := newRouter("GET", "/settings", GetSettings)
	w := doRequest(r, "GET", "/settings", nil, nil)
	if w.Code == http.StatusUnauthorized || w.Code == http.StatusForbidden {
		t.Errorf("GetSettings should be public, got %d", w.Code)
	}
}

// ─── UpdateSettings ───────────────────────────────────────────────────────────

func TestUpdateSettings_MissingExamDate(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/settings",
		setAdminID(adminID.Hex(), "super@test.com", true),
		UpdateSettings,
	)
	w := doRequest(r, "PUT", "/admin/settings", map[string]interface{}{
		"examName": "JNVST 2027",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing examDate, got %d", w.Code)
	}
}

func TestUpdateSettings_MissingExamName(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/settings",
		setAdminID(adminID.Hex(), "super@test.com", true),
		UpdateSettings,
	)
	w := doRequest(r, "PUT", "/admin/settings", map[string]interface{}{
		"examDate": "2027-01-01",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing examName, got %d", w.Code)
	}
}

func TestUpdateSettings_InvalidDateFormat(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/settings",
		setAdminID(adminID.Hex(), "super@test.com", true),
		UpdateSettings,
	)
	w := doRequest(r, "PUT", "/admin/settings", map[string]interface{}{
		"examDate": "01/01/2027", // wrong format
		"examName": "JNVST 2027",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid date format, got %d", w.Code)
	}
}

func TestUpdateSettings_InvalidDate_Letters(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/settings",
		setAdminID(adminID.Hex(), "super@test.com", true),
		UpdateSettings,
	)
	w := doRequest(r, "PUT", "/admin/settings", map[string]interface{}{
		"examDate": "not-a-date",
		"examName": "JNVST 2027",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid date, got %d", w.Code)
	}
}

func TestUpdateSettings_Success_Insert(t *testing.T) {
	requireDB(t)
	dropCollection(t, "settings")

	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/settings",
		setAdminID(adminID.Hex(), "super@test.com", true),
		UpdateSettings,
	)
	w := doRequest(r, "PUT", "/admin/settings", map[string]interface{}{
		"examDate": "2027-04-15",
		"examName": "JNVST 2027",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	if data["examName"] != "JNVST 2027" {
		t.Errorf("examName: want JNVST 2027, got %v", data["examName"])
	}
}

func TestUpdateSettings_Success_Update(t *testing.T) {
	requireDB(t)
	dropCollection(t, "settings")

	adminID := primitive.NewObjectID()
	ctx := context.Background()
	config.GetCollection("settings").InsertOne(ctx, models.Settings{
		ID:       primitive.NewObjectID(),
		ExamDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ExamName: "Old Name",
	})

	r := newRouter("PUT", "/admin/settings",
		setAdminID(adminID.Hex(), "super@test.com", true),
		UpdateSettings,
	)
	w := doRequest(r, "PUT", "/admin/settings", map[string]interface{}{
		"examDate": "2026-06-20",
		"examName": "Updated Name",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSettings_EmptyExamName(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/settings",
		setAdminID(adminID.Hex(), "super@test.com", true),
		UpdateSettings,
	)
	w := doRequest(r, "PUT", "/admin/settings", map[string]interface{}{
		"examDate": "2027-01-01",
		"examName": "",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty examName (required field), got %d", w.Code)
	}
}

func TestUpdateSettings_ValidDate_ISOFormat(t *testing.T) {
	requireDB(t)
	dropCollection(t, "settings")

	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/settings",
		setAdminID(adminID.Hex(), "super@test.com", true),
		UpdateSettings,
	)
	w := doRequest(r, "PUT", "/admin/settings", map[string]interface{}{
		"examDate": "2026-12-31",
		"examName": "Year End Exam",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}
