package handlers

import (
	"net/http"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ─── GetStudentAnalytics ──────────────────────────────────────────────────────

func TestGetStudentAnalytics_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/analytics", setUserID(userID.Hex(), "9876543210"), GetStudentAnalytics)
	w := doRequest(r, "GET", "/analytics", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})

	// All arrays should be empty or zero
	if _, ok := data["subjectAccuracy"]; !ok {
		t.Error("expected subjectAccuracy in response")
	}
	if _, ok := data["scoreTrend"]; !ok {
		t.Error("expected scoreTrend in response")
	}
	if _, ok := data["weakAreas"]; !ok {
		t.Error("expected weakAreas in response")
	}
	if _, ok := data["summary"]; !ok {
		t.Error("expected summary in response")
	}
}

func TestGetStudentAnalytics_SummaryHasExpectedFields(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")
	dropCollection(t, "userchapterprogress")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/analytics", setUserID(userID.Hex(), "9876543210"), GetStudentAnalytics)
	w := doRequest(r, "GET", "/analytics", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	summary := data["summary"].(map[string]interface{})

	fields := []string{"totalAttempts", "overallAccuracy", "bestPercent", "chaptersAttempted"}
	for _, f := range fields {
		if _, ok := summary[f]; !ok {
			t.Errorf("expected %s in summary", f)
		}
	}
}

func TestGetStudentAnalytics_NoAttempts_ZeroSummary(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/analytics", setUserID(userID.Hex(), "9876543210"), GetStudentAnalytics)
	w := doRequest(r, "GET", "/analytics", nil, nil)

	data := parseBody(t, w)["data"].(map[string]interface{})
	summary := data["summary"].(map[string]interface{})
	if summary["totalAttempts"].(float64) != 0 {
		t.Errorf("totalAttempts: want 0, got %v", summary["totalAttempts"])
	}
}
