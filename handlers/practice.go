package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"
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

	correct := 0
	type DetailedAnswer struct {
		QuestionID   primitive.ObjectID `json:"questionId"`
		SelectedIdx  int                `json:"selectedIndex"`
		CorrectIdx   int                `json:"correctIndex"`
		IsCorrect    bool               `json:"isCorrect"`
		Explanation  string             `json:"explanation"`
	}

	detailed := make([]DetailedAnswer, len(questions))
	for i, q := range questions {
		key := string(rune('0' + i))
		selectedIdx, exists := body.Answers[key]
		if !exists {
			selectedIdx = -1
		}
		isCorrect := selectedIdx == q.CorrectIndex
		if isCorrect {
			correct++
		}
		detailed[i] = DetailedAnswer{
			QuestionID:  q.ID,
			SelectedIdx: selectedIdx,
			CorrectIdx:  q.CorrectIndex,
			IsCorrect:   isCorrect,
			Explanation: q.Explanation,
		}
	}

	total := len(questions)
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
