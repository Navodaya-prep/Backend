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

// ─── calculatePoints (pure function) ─────────────────────────────────────────

func TestCalculatePoints_FirstAttempt_UltraFast(t *testing.T) {
	// 5s, attempt 1 → 100 + 50 = 150
	pts := calculatePoints(5000, 1)
	if pts != 150 {
		t.Errorf("want 150, got %d", pts)
	}
}

func TestCalculatePoints_FirstAttempt_ExactlyTenSeconds(t *testing.T) {
	// 10s exactly → speed bonus = 50
	pts := calculatePoints(10000, 1)
	if pts != 150 {
		t.Errorf("want 150, got %d", pts)
	}
}

func TestCalculatePoints_FirstAttempt_SlightlyOverTen(t *testing.T) {
	// 10.1s → timeSec=10.1, speedBonus = round(50*(60-10.1)/50) = round(49.9) = 50
	pts := calculatePoints(10100, 1)
	if pts != 150 {
		t.Errorf("want 150 (≈10s still gives full bonus), got %d", pts)
	}
}

func TestCalculatePoints_FirstAttempt_HalfwayThrough(t *testing.T) {
	// 35s → speedBonus = round(50*(60-35)/50) = round(25) = 25 → 100+25 = 125
	pts := calculatePoints(35000, 1)
	if pts != 125 {
		t.Errorf("want 125, got %d", pts)
	}
}

func TestCalculatePoints_FirstAttempt_JustUnder60s(t *testing.T) {
	// 59.9s → speedBonus = round(50*(60-59.9)/50) = round(0.1) = 0 → 100
	pts := calculatePoints(59900, 1)
	if pts != 100 {
		t.Errorf("want 100 (no speed bonus at ~60s), got %d", pts)
	}
}

func TestCalculatePoints_FirstAttempt_ExactlySixtySeconds(t *testing.T) {
	// 60s exactly → speedBonus = 0 → 100
	pts := calculatePoints(60000, 1)
	if pts != 100 {
		t.Errorf("want 100, got %d", pts)
	}
}

func TestCalculatePoints_FirstAttempt_VeryLate(t *testing.T) {
	// 120s → speedBonus = 0 → 100
	pts := calculatePoints(120000, 1)
	if pts != 100 {
		t.Errorf("want 100 (very late answer, no bonus), got %d", pts)
	}
}

func TestCalculatePoints_FirstAttempt_ZeroTime(t *testing.T) {
	// 0ms → considered ≤10s, full speed bonus
	pts := calculatePoints(0, 1)
	if pts != 150 {
		t.Errorf("want 150, got %d", pts)
	}
}

func TestCalculatePoints_SecondAttempt_Fast(t *testing.T) {
	// 5s, attempt 2 → 100 + 50 - 20 = 130
	pts := calculatePoints(5000, 2)
	if pts != 130 {
		t.Errorf("want 130, got %d", pts)
	}
}

func TestCalculatePoints_ThirdAttempt_Fast(t *testing.T) {
	// 5s, attempt 3 → 100 + 50 - 40 = 110
	pts := calculatePoints(5000, 3)
	if pts != 110 {
		t.Errorf("want 110, got %d", pts)
	}
}

func TestCalculatePoints_FifthAttempt_Fast(t *testing.T) {
	// 5s, attempt 5 → 100 + 50 - 80 = 70
	pts := calculatePoints(5000, 5)
	if pts != 70 {
		t.Errorf("want 70, got %d", pts)
	}
}

func TestCalculatePoints_ManyAttempts_ClampedToMinimum(t *testing.T) {
	// attempt 10, slow → 100 + 0 - 180 = -80 → clamped to 10
	pts := calculatePoints(120000, 10)
	if pts != 10 {
		t.Errorf("want 10 (minimum), got %d", pts)
	}
}

func TestCalculatePoints_SecondAttempt_VerySlow(t *testing.T) {
	// 90s, attempt 2 → 100 + 0 - 20 = 80
	pts := calculatePoints(90000, 2)
	if pts != 80 {
		t.Errorf("want 80, got %d", pts)
	}
}

func TestCalculatePoints_SixthAttempt_Slow(t *testing.T) {
	// 90s, attempt 6 → 100 + 0 - 100 = 0 → clamped to 10
	pts := calculatePoints(90000, 6)
	if pts != 10 {
		t.Errorf("want 10 (minimum), got %d", pts)
	}
}

func TestCalculatePoints_NeverNegative(t *testing.T) {
	// Ensure points never go below 10
	for attempt := 1; attempt <= 20; attempt++ {
		for _, ms := range []int64{0, 5000, 30000, 60000, 120000} {
			pts := calculatePoints(ms, attempt)
			if pts < 10 {
				t.Errorf("points went below 10: attempt=%d ms=%d pts=%d", attempt, ms, pts)
			}
		}
	}
}

// ─── AdminListChallenges ──────────────────────────────────────────────────────

func TestAdminListChallenges_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")
	dropCollection(t, "daily_challenge_attempts")

	r := newRouter("GET", "/admin/daily-challenge",
		setAdminID("admin1", "admin@test.com", true),
		AdminListChallenges,
	)
	w := doRequest(r, "GET", "/admin/daily-challenge", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

func TestAdminListChallenges_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")
	dropCollection(t, "daily_challenge_attempts")

	ctx := context.Background()
	config.GetCollection("daily_challenges").InsertOne(ctx, models.DailyChallenge{
		ID:        primitive.NewObjectID(),
		Date:      "2024-01-01",
		Text:      "Test question",
		Options:   []string{"A", "B", "C", "D"},
		CreatedAt: time.Now(),
	})

	r := newRouter("GET", "/admin/daily-challenge",
		setAdminID("admin1", "admin@test.com", true),
		AdminListChallenges,
	)
	w := doRequest(r, "GET", "/admin/daily-challenge", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminCreateChallenge ─────────────────────────────────────────────────────

func TestAdminCreateChallenge_MissingFields(t *testing.T) {
	r := newRouter("POST", "/admin/daily-challenge",
		setAdminID("admin1", "admin@test.com", true),
		AdminCreateChallenge,
	)
	w := doRequest(r, "POST", "/admin/daily-challenge", map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestAdminCreateChallenge_TooFewOptions(t *testing.T) {
	r := newRouter("POST", "/admin/daily-challenge",
		setAdminID("admin1", "admin@test.com", true),
		AdminCreateChallenge,
	)
	body := map[string]interface{}{
		"date":         "2024-01-15",
		"text":         "Question?",
		"options":      []string{"Only one"},
		"correctIndex": 0,
	}
	w := doRequest(r, "POST", "/admin/daily-challenge", body, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for <2 options, got %d", w.Code)
	}
}

func TestAdminCreateChallenge_InvalidCorrectIndex(t *testing.T) {
	r := newRouter("POST", "/admin/daily-challenge",
		setAdminID("admin1", "admin@test.com", true),
		AdminCreateChallenge,
	)
	body := map[string]interface{}{
		"date":         "2024-01-15",
		"text":         "Question?",
		"options":      []string{"A", "B"},
		"correctIndex": 5, // out of range
	}
	w := doRequest(r, "POST", "/admin/daily-challenge", body, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid correctIndex, got %d", w.Code)
	}
}

func TestAdminCreateChallenge_InvalidDateFormat(t *testing.T) {
	r := newRouter("POST", "/admin/daily-challenge",
		setAdminID("admin1", "admin@test.com", true),
		AdminCreateChallenge,
	)
	body := map[string]interface{}{
		"date":         "01/15/2024", // wrong format
		"text":         "Question?",
		"options":      []string{"A", "B"},
		"correctIndex": 0,
	}
	w := doRequest(r, "POST", "/admin/daily-challenge", body, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid date format, got %d", w.Code)
	}
}

func TestAdminCreateChallenge_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")

	r := newRouter("POST", "/admin/daily-challenge",
		setAdminID("admin1", "admin@test.com", true),
		AdminCreateChallenge,
	)
	body := map[string]interface{}{
		"date":         "2024-01-20",
		"text":         "What is 1+1?",
		"options":      []string{"1", "2", "3", "4"},
		"correctIndex": 1,
		"explanation":  "Basic math",
		"subject":      "Math",
		"difficulty":   "easy",
	}
	w := doRequest(r, "POST", "/admin/daily-challenge", body, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	resp := parseBody(t, w)
	if resp["success"] != true {
		t.Error("expected success=true")
	}
}

func TestAdminCreateChallenge_DuplicateDate(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")

	ctx := context.Background()
	config.GetCollection("daily_challenges").InsertOne(ctx, models.DailyChallenge{
		ID:   primitive.NewObjectID(),
		Date: "2024-02-01",
		Text: "Existing question",
	})

	r := newRouter("POST", "/admin/daily-challenge",
		setAdminID("admin1", "admin@test.com", true),
		AdminCreateChallenge,
	)
	body := map[string]interface{}{
		"date":         "2024-02-01", // same date
		"text":         "Duplicate",
		"options":      []string{"A", "B"},
		"correctIndex": 0,
	}
	w := doRequest(r, "POST", "/admin/daily-challenge", body, nil)

	if w.Code != http.StatusConflict {
		t.Errorf("want 409 for duplicate date, got %d", w.Code)
	}
}

// ─── AdminUpdateChallenge ─────────────────────────────────────────────────────

func TestAdminUpdateChallenge_InvalidID(t *testing.T) {
	r := newRouter("PUT", "/admin/daily-challenge/:id",
		setAdminID("admin1", "admin@test.com", true),
		AdminUpdateChallenge,
	)
	w := doRequest(r, "PUT", "/admin/daily-challenge/not-an-id", map[string]interface{}{
		"date": "2024-01-01", "text": "x", "options": []string{"A", "B"},
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestAdminUpdateChallenge_MissingFields(t *testing.T) {
	id := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/daily-challenge/:id",
		setAdminID("admin1", "admin@test.com", true),
		AdminUpdateChallenge,
	)
	w := doRequest(r, "PUT", "/admin/daily-challenge/"+id.Hex(), map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing fields, got %d", w.Code)
	}
}

func TestAdminUpdateChallenge_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")

	id := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/daily-challenge/:id",
		setAdminID("admin1", "admin@test.com", true),
		AdminUpdateChallenge,
	)
	body := map[string]interface{}{
		"date": "2024-03-01", "text": "Updated", "options": []string{"A", "B"},
	}
	w := doRequest(r, "PUT", "/admin/daily-challenge/"+id.Hex(), body, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent challenge, got %d", w.Code)
	}
}

func TestAdminUpdateChallenge_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")

	ctx := context.Background()
	ch := models.DailyChallenge{
		ID:      primitive.NewObjectID(),
		Date:    "2024-03-10",
		Text:    "Old text",
		Options: []string{"A", "B"},
	}
	config.GetCollection("daily_challenges").InsertOne(ctx, ch)

	r := newRouter("PUT", "/admin/daily-challenge/:id",
		setAdminID("admin1", "admin@test.com", true),
		AdminUpdateChallenge,
	)
	body := map[string]interface{}{
		"date": "2024-03-10", "text": "New text",
		"options": []string{"A", "B", "C"}, "correctIndex": 1,
	}
	w := doRequest(r, "PUT", "/admin/daily-challenge/"+ch.ID.Hex(), body, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── AdminDeleteChallenge ─────────────────────────────────────────────────────

func TestAdminDeleteChallenge_InvalidID(t *testing.T) {
	r := newRouter("DELETE", "/admin/daily-challenge/:id",
		setAdminID("admin1", "admin@test.com", true),
		AdminDeleteChallenge,
	)
	w := doRequest(r, "DELETE", "/admin/daily-challenge/bad-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestAdminDeleteChallenge_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")

	r := newRouter("DELETE", "/admin/daily-challenge/:id",
		setAdminID("admin1", "admin@test.com", true),
		AdminDeleteChallenge,
	)
	w := doRequest(r, "DELETE", "/admin/daily-challenge/"+primitive.NewObjectID().Hex(), nil, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestAdminDeleteChallenge_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")
	dropCollection(t, "daily_challenge_attempts")

	ctx := context.Background()
	ch := models.DailyChallenge{
		ID:   primitive.NewObjectID(),
		Date: "2024-04-01",
		Text: "Delete me",
	}
	config.GetCollection("daily_challenges").InsertOne(ctx, ch)

	r := newRouter("DELETE", "/admin/daily-challenge/:id",
		setAdminID("admin1", "admin@test.com", true),
		AdminDeleteChallenge,
	)
	w := doRequest(r, "DELETE", "/admin/daily-challenge/"+ch.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── GetTodayChallenge ────────────────────────────────────────────────────────

func TestGetTodayChallenge_NoChallengeToday(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/daily-challenge/today",
		setUserID(userID.Hex(), "9876543210"),
		GetTodayChallenge,
	)
	w := doRequest(r, "GET", "/daily-challenge/today", nil, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestGetTodayChallenge_WithChallenge(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")
	dropCollection(t, "daily_challenge_attempts")

	today := time.Now().Format("2006-01-02")
	ctx := context.Background()
	ch := models.DailyChallenge{
		ID:           primitive.NewObjectID(),
		Date:         today,
		Text:         "Today's challenge",
		Options:      []string{"A", "B", "C", "D"},
		CorrectIndex: 2,
		Explanation:  "Because C",
	}
	config.GetCollection("daily_challenges").InsertOne(ctx, ch)

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/daily-challenge/today",
		setUserID(userID.Hex(), "9876543210"),
		GetTodayChallenge,
	)
	w := doRequest(r, "GET", "/daily-challenge/today", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

func TestGetTodayChallenge_HidesAnswerWhenNotAttempted(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")
	dropCollection(t, "daily_challenge_attempts")

	today := time.Now().Format("2006-01-02")
	ctx := context.Background()
	ch := models.DailyChallenge{
		ID:           primitive.NewObjectID(),
		Date:         today,
		Text:         "Hidden answer test",
		Options:      []string{"A", "B"},
		CorrectIndex: 1,
		Explanation:  "Secret",
	}
	config.GetCollection("daily_challenges").InsertOne(ctx, ch)

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/daily-challenge/today",
		setUserID(userID.Hex(), "9876543210"),
		GetTodayChallenge,
	)
	w := doRequest(r, "GET", "/daily-challenge/today", nil, nil)

	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	challenge := data["challenge"].(map[string]interface{})

	// correctIndex and explanation should NOT be present when not attempted
	if _, hasCorrect := challenge["correctIndex"]; hasCorrect {
		t.Error("correctIndex should be hidden when challenge not attempted")
	}
	if _, hasExpl := challenge["explanation"]; hasExpl {
		t.Error("explanation should be hidden when challenge not attempted")
	}
}

// ─── SubmitDailyChallenge ─────────────────────────────────────────────────────

func TestSubmitDailyChallenge_MissingBody(t *testing.T) {
	// Empty/nil body causes ShouldBindJSON EOF → 400 INVALID_INPUT
	userID := primitive.NewObjectID()
	r := newRouter("POST", "/daily-challenge/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitDailyChallenge,
	)
	w := doRequest(r, "POST", "/daily-challenge/submit", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty body, got %d", w.Code)
	}
}

func TestSubmitDailyChallenge_NoChallengeToday(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/daily-challenge/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitDailyChallenge,
	)
	w := doRequest(r, "POST", "/daily-challenge/submit", map[string]interface{}{
		"selectedIndex": 0, "timeTakenMs": 5000,
	}, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 when no challenge today, got %d", w.Code)
	}
}

func TestSubmitDailyChallenge_CorrectAnswer(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")
	dropCollection(t, "daily_challenge_attempts")
	dropCollection(t, "users")

	today := time.Now().Format("2006-01-02")
	userID := primitive.NewObjectID()
	ctx := context.Background()

	config.GetCollection("daily_challenges").InsertOne(ctx, models.DailyChallenge{
		ID:           primitive.NewObjectID(),
		Date:         today,
		Text:         "Correct answer test",
		Options:      []string{"Wrong", "Correct", "Wrong", "Wrong"},
		CorrectIndex: 1,
	})
	config.GetCollection("users").InsertOne(ctx, models.User{
		ID: userID, Name: "Tester", Phone: "9876543210",
	})

	r := newRouter("POST", "/daily-challenge/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitDailyChallenge,
	)
	w := doRequest(r, "POST", "/daily-challenge/submit", map[string]interface{}{
		"selectedIndex": 1, "timeTakenMs": 5000,
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	if data["isCorrect"] != true {
		t.Error("expected isCorrect=true")
	}
}

func TestSubmitDailyChallenge_WrongAnswer(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")
	dropCollection(t, "daily_challenge_attempts")

	today := time.Now().Format("2006-01-02")
	userID := primitive.NewObjectID()
	ctx := context.Background()

	config.GetCollection("daily_challenges").InsertOne(ctx, models.DailyChallenge{
		ID:           primitive.NewObjectID(),
		Date:         today,
		Text:         "Wrong answer test",
		Options:      []string{"A", "B"},
		CorrectIndex: 1,
	})

	r := newRouter("POST", "/daily-challenge/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitDailyChallenge,
	)
	w := doRequest(r, "POST", "/daily-challenge/submit", map[string]interface{}{
		"selectedIndex": 0, "timeTakenMs": 5000,
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	if data["isCorrect"] != false {
		t.Error("expected isCorrect=false")
	}
	if data["points"].(float64) != 0 {
		t.Errorf("expected 0 points for wrong answer, got %v", data["points"])
	}
}

func TestSubmitDailyChallenge_AlreadyCompleted(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")
	dropCollection(t, "daily_challenge_attempts")

	today := time.Now().Format("2006-01-02")
	userID := primitive.NewObjectID()
	ctx := context.Background()

	ch := models.DailyChallenge{
		ID: primitive.NewObjectID(), Date: today,
		Text: "Already done", Options: []string{"A", "B"}, CorrectIndex: 0,
	}
	config.GetCollection("daily_challenges").InsertOne(ctx, ch)

	now := time.Now()
	config.GetCollection("daily_challenge_attempts").InsertOne(ctx, models.DailyChallengeAttempt{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		ChallengeID: ch.ID,
		Date:        today,
		IsCorrect:   true,
		SolvedAt:    &now,
		CreatedAt:   now,
		UpdatedAt:   now,
	})

	r := newRouter("POST", "/daily-challenge/submit",
		setUserID(userID.Hex(), "9876543210"),
		SubmitDailyChallenge,
	)
	w := doRequest(r, "POST", "/daily-challenge/submit", map[string]interface{}{
		"selectedIndex": 0, "timeTakenMs": 1000,
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for already completed, got %d", w.Code)
	}
}

// ─── RevealDailyChallenge ─────────────────────────────────────────────────────

func TestRevealDailyChallenge_NoChallengeToday(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")

	userID := primitive.NewObjectID()
	r := newRouter("POST", "/daily-challenge/reveal",
		setUserID(userID.Hex(), "9876543210"),
		RevealDailyChallenge,
	)
	w := doRequest(r, "POST", "/daily-challenge/reveal", nil, nil)

	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestRevealDailyChallenge_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")
	dropCollection(t, "daily_challenge_attempts")

	today := time.Now().Format("2006-01-02")
	userID := primitive.NewObjectID()
	ctx := context.Background()

	config.GetCollection("daily_challenges").InsertOne(ctx, models.DailyChallenge{
		ID:           primitive.NewObjectID(),
		Date:         today,
		Text:         "Reveal test",
		Options:      []string{"A", "B"},
		CorrectIndex: 0,
		Explanation:  "A is correct",
	})

	r := newRouter("POST", "/daily-challenge/reveal",
		setUserID(userID.Hex(), "9876543210"),
		RevealDailyChallenge,
	)
	w := doRequest(r, "POST", "/daily-challenge/reveal", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	if data["points"].(float64) != 0 {
		t.Errorf("expected 0 points on reveal, got %v", data["points"])
	}
}

func TestRevealDailyChallenge_AlreadyRevealed(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")
	dropCollection(t, "daily_challenge_attempts")

	today := time.Now().Format("2006-01-02")
	userID := primitive.NewObjectID()
	ctx := context.Background()

	ch := models.DailyChallenge{
		ID: primitive.NewObjectID(), Date: today,
		Options: []string{"A", "B"},
	}
	config.GetCollection("daily_challenges").InsertOne(ctx, ch)
	config.GetCollection("daily_challenge_attempts").InsertOne(ctx, models.DailyChallengeAttempt{
		ID: primitive.NewObjectID(), UserID: userID, ChallengeID: ch.ID,
		Date: today, Revealed: true, CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	r := newRouter("POST", "/daily-challenge/reveal",
		setUserID(userID.Hex(), "9876543210"),
		RevealDailyChallenge,
	)
	w := doRequest(r, "POST", "/daily-challenge/reveal", nil, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for already revealed, got %d", w.Code)
	}
}

// ─── GetDailyChallengePractice ────────────────────────────────────────────────

func TestGetDailyChallengePractice_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/daily-challenge/practice",
		setUserID(userID.Hex(), "9876543210"),
		GetDailyChallengePractice,
	)
	w := doRequest(r, "GET", "/daily-challenge/practice", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestGetDailyChallengePractice_WithPastChallenges(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenges")
	dropCollection(t, "daily_challenge_attempts")

	ctx := context.Background()
	// Insert past challenges (not today)
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
	twoDaysAgo := time.Now().AddDate(0, 0, -2).Format("2006-01-02")

	config.GetCollection("daily_challenges").InsertOne(ctx, models.DailyChallenge{
		ID: primitive.NewObjectID(), Date: yesterday,
		Text: "Yesterday", Options: []string{"A", "B"}, CorrectIndex: 0,
	})
	config.GetCollection("daily_challenges").InsertOne(ctx, models.DailyChallenge{
		ID: primitive.NewObjectID(), Date: twoDaysAgo,
		Text: "Two days ago", Options: []string{"A", "B"}, CorrectIndex: 1,
	})

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/daily-challenge/practice",
		setUserID(userID.Hex(), "9876543210"),
		GetDailyChallengePractice,
	)
	w := doRequest(r, "GET", "/daily-challenge/practice", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

// ─── GetDailyChallengeLeaderboard ─────────────────────────────────────────────

func TestGetDailyChallengeLeaderboard_Today(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenge_attempts")
	dropCollection(t, "users")

	userID := primitive.NewObjectID()
	r := newRouter("GET", "/daily-challenge/leaderboard",
		setUserID(userID.Hex(), "9876543210"),
		GetDailyChallengeLeaderboard,
	)
	w := doRequest(r, "GET", "/daily-challenge/leaderboard", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetDailyChallengeLeaderboard_Month(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenge_attempts")
	dropCollection(t, "users")

	userID := primitive.NewObjectID()
	// Register with a wildcard path so query params are preserved
	r := newRouter("GET", "/daily-challenge/leaderboard",
		setUserID(userID.Hex(), "9876543210"),
		GetDailyChallengeLeaderboard,
	)
	// doRequest path can include query string
	w := doRequest(r, "GET", "/daily-challenge/leaderboard?period=month", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
}

func TestGetDailyChallengeLeaderboard_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenge_attempts")
	dropCollection(t, "users")

	userID := primitive.NewObjectID()
	today := time.Now().Format("2006-01-02")
	ctx := context.Background()

	config.GetCollection("daily_challenge_attempts").InsertOne(ctx, models.DailyChallengeAttempt{
		ID:          primitive.NewObjectID(),
		UserID:      userID,
		ChallengeID: primitive.NewObjectID(),
		Date:        today,
		IsCorrect:   true,
		Points:      100,
		TimeTakenMs: 5000,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	r := newRouter("GET", "/daily-challenge/leaderboard",
		setUserID(userID.Hex(), "9876543210"),
		GetDailyChallengeLeaderboard,
	)
	w := doRequest(r, "GET", "/daily-challenge/leaderboard", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	entries, ok := data["leaderboard"].([]interface{})
	if !ok {
		t.Fatal("expected leaderboard array")
	}
	if len(entries) == 0 {
		t.Error("expected at least 1 leaderboard entry")
	}
}

func TestGetDailyChallengeLeaderboard_UserInTopRank(t *testing.T) {
	requireDB(t)
	dropCollection(t, "daily_challenge_attempts")
	dropCollection(t, "users")

	userID := primitive.NewObjectID()
	today := time.Now().Format("2006-01-02")
	ctx := context.Background()

	config.GetCollection("daily_challenge_attempts").InsertOne(ctx, models.DailyChallengeAttempt{
		ID: primitive.NewObjectID(), UserID: userID,
		ChallengeID: primitive.NewObjectID(),
		Date:        today, IsCorrect: true, Points: 150, TimeTakenMs: 3000,
		CreatedAt: time.Now(), UpdatedAt: time.Now(),
	})

	r := newRouter("GET", "/daily-challenge/leaderboard",
		setUserID(userID.Hex(), "9876543210"),
		GetDailyChallengeLeaderboard,
	)
	w := doRequest(r, "GET", "/daily-challenge/leaderboard", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	// When user appears in leaderboard, userRank field should be set
	if _, ok := data["userRank"]; !ok {
		t.Error("expected userRank field when user appears in leaderboard")
	}
}
