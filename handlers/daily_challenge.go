package handlers

import (
	"context"
	"math"
	"net/http"
	"time"

	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func challengesColl() *mongo.Collection { return config.GetCollection("daily_challenges") }
func attemptsColl() *mongo.Collection   { return config.GetCollection("daily_challenge_attempts") }

// ──────────────────────────────────────────────────────────────────────────────
// Admin endpoints (Super Admin only)
// ──────────────────────────────────────────────────────────────────────────────

// AdminListChallenges returns all daily challenges, most recent first
func AdminListChallenges(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}})
	cursor, err := challengesColl().Find(ctx, bson.M{}, opts)
	if err != nil {
		utils.ErrorRes(c, 500, "DB_ERROR", "Failed to fetch challenges")
		return
	}
	defer cursor.Close(ctx)

	var challenges []models.DailyChallenge
	if err := cursor.All(ctx, &challenges); err != nil {
		utils.ErrorRes(c, 500, "DB_ERROR", "Failed to decode challenges")
		return
	}
	if challenges == nil {
		challenges = []models.DailyChallenge{}
	}

	// Fetch attempt counts per challenge
	type countResult struct {
		ID    primitive.ObjectID `bson:"_id"`
		Count int                `bson:"count"`
	}
	pipeline := mongo.Pipeline{
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$challengeId"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
		}}},
	}
	countCursor, err := attemptsColl().Aggregate(ctx, pipeline)
	countMap := map[primitive.ObjectID]int{}
	if err == nil {
		var counts []countResult
		if countCursor.All(ctx, &counts) == nil {
			for _, c := range counts {
				countMap[c.ID] = c.Count
			}
		}
	}

	type challengeWithStats struct {
		models.DailyChallenge `bson:",inline"`
		AttemptCount          int `json:"attemptCount"`
	}
	result := make([]challengeWithStats, len(challenges))
	for i, ch := range challenges {
		result[i] = challengeWithStats{DailyChallenge: ch, AttemptCount: countMap[ch.ID]}
	}

	utils.Success(c, http.StatusOK, gin.H{"challenges": result}, "Challenges fetched")
}

// AdminCreateChallenge creates a new daily challenge question
func AdminCreateChallenge(c *gin.Context) {
	var body struct {
		Date         string   `json:"date" binding:"required"`
		Text         string   `json:"text" binding:"required"`
		Options      []string `json:"options" binding:"required"`
		CorrectIndex int      `json:"correctIndex"`
		Explanation  string   `json:"explanation"`
		Subject      string   `json:"subject"`
		Difficulty   string   `json:"difficulty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, 400, "INVALID_INPUT", "Date, text, and options are required")
		return
	}
	if len(body.Options) < 2 {
		utils.ErrorRes(c, 400, "INVALID_INPUT", "At least 2 options are required")
		return
	}
	if body.CorrectIndex < 0 || body.CorrectIndex >= len(body.Options) {
		utils.ErrorRes(c, 400, "INVALID_INPUT", "Invalid correct index")
		return
	}

	// Validate date format
	if _, err := time.Parse("2006-01-02", body.Date); err != nil {
		utils.ErrorRes(c, 400, "INVALID_INPUT", "Date must be in YYYY-MM-DD format")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check if challenge already exists for this date
	count, _ := challengesColl().CountDocuments(ctx, bson.M{"date": body.Date})
	if count > 0 {
		utils.ErrorRes(c, 409, "DUPLICATE", "A challenge already exists for this date")
		return
	}

	adminEmail, _ := c.Get("adminEmail")
	challenge := models.DailyChallenge{
		Date:         body.Date,
		Text:         body.Text,
		Options:      body.Options,
		CorrectIndex: body.CorrectIndex,
		Explanation:  body.Explanation,
		Subject:      body.Subject,
		Difficulty:   body.Difficulty,
		CreatedBy:    adminEmail.(string),
		CreatedAt:    time.Now(),
	}

	result, err := challengesColl().InsertOne(ctx, challenge)
	if err != nil {
		utils.ErrorRes(c, 500, "DB_ERROR", "Failed to create challenge")
		return
	}
	challenge.ID = result.InsertedID.(primitive.ObjectID)
	utils.Success(c, http.StatusCreated, gin.H{"challenge": challenge}, "Challenge created")
}

// AdminUpdateChallenge updates an existing daily challenge
func AdminUpdateChallenge(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, 400, "INVALID_ID", "Invalid challenge ID")
		return
	}

	var body struct {
		Date         string   `json:"date" binding:"required"`
		Text         string   `json:"text" binding:"required"`
		Options      []string `json:"options" binding:"required"`
		CorrectIndex int      `json:"correctIndex"`
		Explanation  string   `json:"explanation"`
		Subject      string   `json:"subject"`
		Difficulty   string   `json:"difficulty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, 400, "INVALID_INPUT", "Date, text, and options are required")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Check no other challenge exists for this date
	var existing models.DailyChallenge
	err = challengesColl().FindOne(ctx, bson.M{"date": body.Date, "_id": bson.M{"$ne": id}}).Decode(&existing)
	if err == nil {
		utils.ErrorRes(c, 409, "DUPLICATE", "Another challenge already exists for this date")
		return
	}

	update := bson.M{
		"$set": bson.M{
			"date":         body.Date,
			"text":         body.Text,
			"options":      body.Options,
			"correctIndex": body.CorrectIndex,
			"explanation":  body.Explanation,
			"subject":      body.Subject,
			"difficulty":   body.Difficulty,
		},
	}

	res, err := challengesColl().UpdateByID(ctx, id, update)
	if err != nil || res.MatchedCount == 0 {
		utils.ErrorRes(c, 404, "NOT_FOUND", "Challenge not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Challenge updated")
}

// AdminDeleteChallenge deletes a challenge and all its attempts
func AdminDeleteChallenge(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, 400, "INVALID_ID", "Invalid challenge ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := challengesColl().DeleteOne(ctx, bson.M{"_id": id})
	if err != nil || res.DeletedCount == 0 {
		utils.ErrorRes(c, 404, "NOT_FOUND", "Challenge not found")
		return
	}

	// Clean up attempts
	attemptsColl().DeleteMany(ctx, bson.M{"challengeId": id})

	utils.Success(c, http.StatusOK, nil, "Challenge deleted")
}

// ──────────────────────────────────────────────────────────────────────────────
// Student endpoints
// ──────────────────────────────────────────────────────────────────────────────

// GetTodayChallenge returns today's challenge for the student
func GetTodayChallenge(c *gin.Context) {
	today := time.Now().Format("2006-01-02")
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var challenge models.DailyChallenge
	err := challengesColl().FindOne(ctx, bson.M{"date": today}).Decode(&challenge)
	if err != nil {
		utils.ErrorRes(c, 404, "NOT_FOUND", "No challenge available for today")
		return
	}

	// Check if user has an attempt
	var attempt models.DailyChallengeAttempt
	attemptErr := attemptsColl().FindOne(ctx, bson.M{
		"userId":      userID,
		"challengeId": challenge.ID,
	}).Decode(&attempt)

	hasAttempt := attemptErr == nil

	// Build response — hide correct answer if not solved and not revealed
	resp := gin.H{
		"id":         challenge.ID,
		"date":       challenge.Date,
		"text":       challenge.Text,
		"options":    challenge.Options,
		"subject":    challenge.Subject,
		"difficulty": challenge.Difficulty,
	}

	if hasAttempt {
		resp["attempt"] = gin.H{
			"id":            attempt.ID,
			"selectedIndex": attempt.SelectedIndex,
			"isCorrect":     attempt.IsCorrect,
			"points":        attempt.Points,
			"attempts":      attempt.Attempts,
			"revealed":      attempt.Revealed,
			"timeTakenMs":   attempt.TimeTakenMs,
			"solvedAt":      attempt.SolvedAt,
		}
		// Show correct answer and explanation if solved or revealed
		if attempt.IsCorrect || attempt.Revealed {
			resp["correctIndex"] = challenge.CorrectIndex
			resp["explanation"] = challenge.Explanation
		}
	}

	utils.Success(c, http.StatusOK, gin.H{"challenge": resp}, "Today's challenge")
}

// SubmitDailyChallenge handles a student's answer submission
//
// Scoring:
// - Base: 100 points
// - Speed bonus: up to +50 points for solving within the first 10 seconds, linearly decreasing to 0 over 60 seconds
// - Penalty: -20 points per incorrect attempt (starting from 2nd attempt)
// - Minimum: 10 points for a correct answer
// - Wrong answer: 0 points
// - Revealed answer: 0 points
func SubmitDailyChallenge(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	var body struct {
		SelectedIndex int   `json:"selectedIndex"`
		TimeTakenMs   int64 `json:"timeTakenMs"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, 400, "INVALID_INPUT", "selectedIndex is required")
		return
	}

	today := time.Now().Format("2006-01-02")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Get today's challenge
	var challenge models.DailyChallenge
	err := challengesColl().FindOne(ctx, bson.M{"date": today}).Decode(&challenge)
	if err != nil {
		utils.ErrorRes(c, 404, "NOT_FOUND", "No challenge available for today")
		return
	}

	// Check existing attempt
	var existing models.DailyChallengeAttempt
	attemptErr := attemptsColl().FindOne(ctx, bson.M{
		"userId":      userID,
		"challengeId": challenge.ID,
	}).Decode(&existing)

	hasExisting := attemptErr == nil

	// If already solved or revealed, reject
	if hasExisting && (existing.IsCorrect || existing.Revealed) {
		utils.ErrorRes(c, 400, "ALREADY_COMPLETED", "You have already completed today's challenge")
		return
	}

	isCorrect := body.SelectedIndex == challenge.CorrectIndex
	attemptNum := 1
	if hasExisting {
		attemptNum = existing.Attempts + 1
	}

	// Calculate points
	points := 0
	if isCorrect {
		points = calculatePoints(body.TimeTakenMs, attemptNum)
	}

	now := time.Now()

	if hasExisting {
		// Update existing attempt
		update := bson.M{
			"$set": bson.M{
				"selectedIndex": body.SelectedIndex,
				"isCorrect":     isCorrect,
				"points":        points,
				"attempts":      attemptNum,
				"updatedAt":     now,
			},
		}
		if isCorrect {
			update["$set"].(bson.M)["solvedAt"] = now
			update["$set"].(bson.M)["timeTakenMs"] = body.TimeTakenMs
		}
		attemptsColl().UpdateByID(ctx, existing.ID, update)
	} else {
		// Create new attempt
		attempt := models.DailyChallengeAttempt{
			UserID:        userID,
			ChallengeID:   challenge.ID,
			Date:          today,
			SelectedIndex: body.SelectedIndex,
			IsCorrect:     isCorrect,
			Points:        points,
			Attempts:      1,
			TimeTakenMs:   body.TimeTakenMs,
			CreatedAt:     now,
			UpdatedAt:     now,
		}
		if isCorrect {
			attempt.SolvedAt = &now
		}
		attemptsColl().InsertOne(ctx, attempt)
	}

	// Update user's starPoints if correct
	if isCorrect {
		usersColl := config.GetCollection("users")
		usersColl.UpdateByID(ctx, userID, bson.M{
			"$inc": bson.M{"starPoints": points},
		})
	}

	resp := gin.H{
		"isCorrect": isCorrect,
		"points":    points,
		"attempts":  attemptNum,
	}
	if isCorrect {
		resp["correctIndex"] = challenge.CorrectIndex
		resp["explanation"] = challenge.Explanation
	}

	utils.Success(c, http.StatusOK, resp, "Answer submitted")
}

// RevealDailyChallenge reveals the answer without solving - gives 0 points
func RevealDailyChallenge(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))
	today := time.Now().Format("2006-01-02")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var challenge models.DailyChallenge
	err := challengesColl().FindOne(ctx, bson.M{"date": today}).Decode(&challenge)
	if err != nil {
		utils.ErrorRes(c, 404, "NOT_FOUND", "No challenge available for today")
		return
	}

	// Check existing attempt
	var existing models.DailyChallengeAttempt
	attemptErr := attemptsColl().FindOne(ctx, bson.M{
		"userId":      userID,
		"challengeId": challenge.ID,
	}).Decode(&existing)

	if attemptErr == nil && (existing.IsCorrect || existing.Revealed) {
		utils.ErrorRes(c, 400, "ALREADY_COMPLETED", "Already completed or revealed")
		return
	}

	now := time.Now()
	if attemptErr == nil {
		// Update existing
		attemptsColl().UpdateByID(ctx, existing.ID, bson.M{
			"$set": bson.M{
				"revealed":  true,
				"points":    0,
				"updatedAt": now,
			},
		})
	} else {
		// Create revealed attempt
		attempt := models.DailyChallengeAttempt{
			UserID:      userID,
			ChallengeID: challenge.ID,
			Date:        today,
			Revealed:    true,
			Points:      0,
			Attempts:    0,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		attemptsColl().InsertOne(ctx, attempt)
	}

	utils.Success(c, http.StatusOK, gin.H{
		"correctIndex": challenge.CorrectIndex,
		"explanation":  challenge.Explanation,
		"points":       0,
	}, "Answer revealed")
}

// GetDailyChallengeLeaderboard returns top 10 + user rank for today or current month
func GetDailyChallengeLeaderboard(c *gin.Context) {
	period := c.DefaultQuery("period", "today") // "today" or "month"
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var dateFilter bson.M
	now := time.Now()
	if period == "month" {
		// Current month: YYYY-MM prefix
		prefix := now.Format("2006-01")
		dateFilter = bson.M{"date": bson.M{"$regex": "^" + prefix}}
	} else {
		// Today
		dateFilter = bson.M{"date": now.Format("2006-01-02")}
	}

	// Aggregate: sum points per user, sort desc, limit top 10
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: dateFilter}},
		{{Key: "$match", Value: bson.M{"points": bson.M{"$gt": 0}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$userId"},
			{Key: "totalPoints", Value: bson.D{{Key: "$sum", Value: "$points"}}},
			{Key: "challengesSolved", Value: bson.D{{Key: "$sum", Value: bson.D{
				{Key: "$cond", Value: bson.A{bson.D{{Key: "$eq", Value: bson.A{"$isCorrect", true}}}, 1, 0}},
			}}}},
			{Key: "fastestTime", Value: bson.D{{Key: "$min", Value: "$timeTakenMs"}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "totalPoints", Value: -1}, {Key: "fastestTime", Value: 1}}}},
	}

	cursor, err := attemptsColl().Aggregate(ctx, pipeline)
	if err != nil {
		utils.ErrorRes(c, 500, "DB_ERROR", "Failed to fetch leaderboard")
		return
	}
	defer cursor.Close(ctx)

	type aggResult struct {
		UserID           primitive.ObjectID `bson:"_id"`
		TotalPoints      int                `bson:"totalPoints"`
		ChallengesSolved int                `bson:"challengesSolved"`
		FastestTime      int64              `bson:"fastestTime"`
	}
	var allResults []aggResult
	if err := cursor.All(ctx, &allResults); err != nil {
		utils.ErrorRes(c, 500, "DB_ERROR", "Failed to decode leaderboard")
		return
	}

	// Get top 10
	top := allResults
	if len(top) > 10 {
		top = top[:10]
	}

	// Collect user IDs for lookup
	userIDs := make([]primitive.ObjectID, len(top))
	for i, r := range top {
		userIDs[i] = r.UserID
	}

	// Also check if current user is in top 10
	userRank := 0
	var userEntry *aggResult
	for i, r := range allResults {
		if r.UserID == userID {
			userRank = i + 1
			userEntry = &allResults[i]
			// Add user to lookup if not in top 10
			found := false
			for _, uid := range userIDs {
				if uid == userID {
					found = true
					break
				}
			}
			if !found {
				userIDs = append(userIDs, userID)
			}
			break
		}
	}

	// Lookup user details
	usersColl := config.GetCollection("users")
	userCursor, _ := usersColl.Find(ctx, bson.M{"_id": bson.M{"$in": userIDs}})
	type userInfo struct {
		ID    primitive.ObjectID `bson:"_id"`
		Name  string             `bson:"name"`
		State string             `bson:"state"`
	}
	userMap := map[primitive.ObjectID]userInfo{}
	if userCursor != nil {
		var users []userInfo
		if userCursor.All(ctx, &users) == nil {
			for _, u := range users {
				userMap[u.ID] = u
			}
		}
	}

	// Build response
	type leaderboardEntry struct {
		Rank             int    `json:"rank"`
		UserID           string `json:"userId"`
		Name             string `json:"name"`
		State            string `json:"state"`
		Avatar           string `json:"avatar"`
		TotalPoints      int    `json:"totalPoints"`
		ChallengesSolved int    `json:"challengesSolved"`
		FastestTime      int64  `json:"fastestTime"`
	}

	entries := make([]leaderboardEntry, len(top))
	for i, r := range top {
		u := userMap[r.UserID]
		avatar := "?"
		if len(u.Name) > 0 {
			avatar = string([]rune(u.Name)[0])
		}
		entries[i] = leaderboardEntry{
			Rank:             i + 1,
			UserID:           r.UserID.Hex(),
			Name:             u.Name,
			State:            u.State,
			Avatar:           avatar,
			TotalPoints:      r.TotalPoints,
			ChallengesSolved: r.ChallengesSolved,
			FastestTime:      r.FastestTime,
		}
	}

	resp := gin.H{
		"leaderboard": entries,
		"period":      period,
	}

	if userEntry != nil {
		u := userMap[userID]
		avatar := "?"
		if len(u.Name) > 0 {
			avatar = string([]rune(u.Name)[0])
		}
		resp["userRank"] = leaderboardEntry{
			Rank:             userRank,
			UserID:           userID.Hex(),
			Name:             u.Name,
			State:            u.State,
			Avatar:           avatar,
			TotalPoints:      userEntry.TotalPoints,
			ChallengesSolved: userEntry.ChallengesSolved,
			FastestTime:      userEntry.FastestTime,
		}
	}

	utils.Success(c, http.StatusOK, resp, "Leaderboard fetched")
}

// GetDailyChallengePractice returns all past daily challenges for practice
func GetDailyChallengePractice(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	today := time.Now().Format("2006-01-02")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Fetch past challenges (not today)
	opts := options.Find().SetSort(bson.D{{Key: "date", Value: -1}}).SetLimit(50)
	cursor, err := challengesColl().Find(ctx, bson.M{"date": bson.M{"$lt": today}}, opts)
	if err != nil {
		utils.ErrorRes(c, 500, "DB_ERROR", "Failed to fetch challenges")
		return
	}
	defer cursor.Close(ctx)

	var challenges []models.DailyChallenge
	if err := cursor.All(ctx, &challenges); err != nil {
		utils.ErrorRes(c, 500, "DB_ERROR", "Failed to decode challenges")
		return
	}
	if challenges == nil {
		challenges = []models.DailyChallenge{}
	}

	// Get user's attempts for these challenges
	challengeIDs := make([]primitive.ObjectID, len(challenges))
	for i, ch := range challenges {
		challengeIDs[i] = ch.ID
	}

	attemptCursor, _ := attemptsColl().Find(ctx, bson.M{
		"userId":      userID,
		"challengeId": bson.M{"$in": challengeIDs},
	})

	attemptMap := map[primitive.ObjectID]models.DailyChallengeAttempt{}
	if attemptCursor != nil {
		var attempts []models.DailyChallengeAttempt
		if attemptCursor.All(ctx, &attempts) == nil {
			for _, a := range attempts {
				attemptMap[a.ChallengeID] = a
			}
		}
	}

	type practiceChallenge struct {
		ID           primitive.ObjectID `json:"id"`
		Date         string             `json:"date"`
		Text         string             `json:"text"`
		Options      []string           `json:"options"`
		CorrectIndex int                `json:"correctIndex"`
		Explanation  string             `json:"explanation"`
		Subject      string             `json:"subject"`
		Difficulty   string             `json:"difficulty"`
		UserSolved   bool               `json:"userSolved"`
		UserPoints   int                `json:"userPoints"`
	}

	result := make([]practiceChallenge, len(challenges))
	for i, ch := range challenges {
		attempt, hasAttempt := attemptMap[ch.ID]
		result[i] = practiceChallenge{
			ID:           ch.ID,
			Date:         ch.Date,
			Text:         ch.Text,
			Options:      ch.Options,
			CorrectIndex: ch.CorrectIndex,
			Explanation:  ch.Explanation,
			Subject:      ch.Subject,
			Difficulty:   ch.Difficulty,
		}
		if hasAttempt {
			result[i].UserSolved = attempt.IsCorrect || attempt.Revealed
			result[i].UserPoints = attempt.Points
		}
	}

	utils.Success(c, http.StatusOK, gin.H{"challenges": result}, "Practice challenges fetched")
}

// ──────────────────────────────────────────────────────────────────────────────
// Scoring helpers
// ──────────────────────────────────────────────────────────────────────────────

// calculatePoints returns points for a correct answer based on speed and attempt number.
//
//	Base: 100 points
//	Speed bonus: up to +50 if solved within 10s, linearly decreasing to 0 at 60s
//	Penalty: -20 per additional attempt (starting from attempt 2)
//	Minimum: 10 points
func calculatePoints(timeTakenMs int64, attemptNum int) int {
	base := 100

	// Speed bonus: 50 points max for ≤10s, linear decrease to 0 at ≥60s
	speedBonus := 0
	timeSec := float64(timeTakenMs) / 1000.0
	if timeSec <= 10 {
		speedBonus = 50
	} else if timeSec < 60 {
		speedBonus = int(math.Round(50 * (60 - timeSec) / 50))
	}

	// Penalty for reattempts (attempt 1 = no penalty)
	penalty := 0
	if attemptNum > 1 {
		penalty = (attemptNum - 1) * 20
	}

	points := base + speedBonus - penalty
	if points < 10 {
		points = 10
	}
	return points
}
