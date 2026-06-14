package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"github.com/navodayasarthi/api/config"
	"github.com/navodayasarthi/api/models"
	"github.com/navodayasarthi/api/utils"
)

const freeQuestionsPerChapter = 10

func GetPracticeQuestions(c *gin.Context) {
	chapterID, err := primitive.ObjectIDFromHex(c.Param("chapterId"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid chapter ID")
		return
	}

	col := config.GetCollection("questions")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find().SetSort(bson.M{"difficulty": 1}).SetLimit(freeQuestionsPerChapter)
	cursor, err := col.Find(ctx, bson.M{"chapterId": chapterID}, opts)
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch questions")
		return
	}
	defer cursor.Close(ctx)

	var questions []models.Question
	if err := cursor.All(ctx, &questions); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "DECODE_FAILED", "Failed to decode questions")
		return
	}
	if questions == nil {
		questions = []models.Question{}
	}

	// Hide answer data for premium questions from non-premium users.
	if userIDStr, ok := c.Get("userId"); ok {
		if userID, err := primitive.ObjectIDFromHex(userIDStr.(string)); err == nil {
			redactPremiumQuestions(questions, userIsPremium(userID))
		}
	}

	utils.Success(c, http.StatusOK, gin.H{"questions": questions}, "Success")
}

func SubmitPractice(c *gin.Context) {
	var body struct {
		ChapterID string         `json:"chapterId" binding:"required"`
		Answers   map[string]int `json:"answers" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "ChapterId and answers are required")
		return
	}

	chapterID, err := primitive.ObjectIDFromHex(body.ChapterID)
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid chapter ID")
		return
	}

	col := config.GetCollection("questions")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := col.Find(ctx, bson.M{"chapterId": chapterID})
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch questions")
		return
	}
	defer cursor.Close(ctx)

	var questions []models.Question
	cursor.All(ctx, &questions)

	// Non-premium users cannot submit premium questions — they are neither
	// scored nor revealed (gate can't be bypassed via direct POST).
	isPremium := false
	if userIDStr, ok := c.Get("userId"); ok {
		if userID, err := primitive.ObjectIDFromHex(userIDStr.(string)); err == nil {
			isPremium = userIsPremium(userID)
		}
	}

	correct := 0
	type DetailedAnswer struct {
		QuestionID   primitive.ObjectID `json:"questionId"`
		SelectedIdx  int                `json:"selectedIndex"`
		CorrectIdx   int                `json:"correctIndex"`
		IsCorrect    bool               `json:"isCorrect"`
		Explanation  string             `json:"explanation"`
	}

	// Preserve the client's positional answer keys ('0'+i) by iterating the
	// original order; skipped (locked) questions just don't produce a result.
	detailed := make([]DetailedAnswer, 0, len(questions))
	total := 0
	for i, q := range questions {
		if !isPremium && q.IsPremium {
			continue
		}
		key := string(rune('0' + i))
		selectedIdx, exists := body.Answers[key]
		if !exists {
			selectedIdx = -1
		}
		isCorrect := selectedIdx == q.CorrectIndex
		if isCorrect {
			correct++
		}
		total++
		detailed = append(detailed, DetailedAnswer{
			QuestionID:  q.ID,
			SelectedIdx: selectedIdx,
			CorrectIdx:  q.CorrectIndex,
			IsCorrect:   isCorrect,
			Explanation: q.Explanation,
		})
	}

	percent := 0
	if total > 0 {
		percent = (correct * 100) / total
	}

	utils.Success(c, http.StatusOK, gin.H{
		"result": gin.H{
			"correct":  correct,
			"total":    total,
			"percent":  percent,
			"detailed": detailed,
		},
	}, "Practice submitted successfully")
}
