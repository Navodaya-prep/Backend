package handlers

import (
	"net/http"
	"testing"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ─── GetLeaderboard ───────────────────────────────────────────────────────────

func TestGetLeaderboard_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/leaderboard", setUserID(userID.Hex(), "9876543210"), GetLeaderboard)
	w := doRequest(r, "GET", "/leaderboard", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	leaderboard, ok := data["leaderboard"].([]interface{})
	if !ok {
		t.Fatal("expected leaderboard array in response")
	}
	if len(leaderboard) != 0 {
		t.Errorf("expected empty leaderboard, got %d entries", len(leaderboard))
	}
}

func TestGetLeaderboard_UserRankMinusOneWhenNotOnBoard(t *testing.T) {
	requireDB(t)
	dropCollection(t, "mocktestsattempts")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/leaderboard", setUserID(userID.Hex(), "9876543210"), GetLeaderboard)
	w := doRequest(r, "GET", "/leaderboard", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	if data["userRank"].(float64) != -1 {
		t.Errorf("userRank: want -1 for user not on board, got %v", data["userRank"])
	}
}

func TestGetLeaderboard_ReturnsOK(t *testing.T) {
	requireDB(t)

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/leaderboard", setUserID(userID.Hex(), "9876543210"), GetLeaderboard)
	w := doRequest(r, "GET", "/leaderboard", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}
