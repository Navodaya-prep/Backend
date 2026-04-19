package handlers

import (
	"context"
	"net/http"
	"time"

	"navodaya-api/config"
	"navodaya-api/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GetStudentAnalytics — GET /analytics
// Returns subject-wise accuracy, mock test trend, and weak areas
func GetStudentAnalytics(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// ── 1. Subject-wise accuracy from mock test attempts ──────────────────────
	subjectPipeline := bson.A{
		bson.M{"$match": bson.M{"userId": userID}},
		bson.M{"$unwind": "$answers"},
		bson.M{
			"$lookup": bson.M{
				"from":         "questions",
				"localField":   "answers.questionId",
				"foreignField": "_id",
				"as":           "q",
			},
		},
		bson.M{"$unwind": bson.M{"path": "$q", "preserveNullAndEmptyArrays": true}},
		bson.M{
			"$group": bson.M{
				"_id": bson.M{
					"$ifNull": bson.A{"$q.subject", "Other"},
				},
				"correct": bson.M{"$sum": bson.M{"$cond": bson.A{"$answers.isCorrect", 1, 0}}},
				"total":   bson.M{"$sum": 1},
			},
		},
		bson.M{
			"$project": bson.M{
				"_id":      0,
				"subject":  "$_id",
				"correct":  1,
				"total":    1,
				"accuracy": bson.M{"$round": bson.A{bson.M{"$multiply": bson.A{bson.M{"$divide": bson.A{"$correct", "$total"}}, 100}}, 1}},
			},
		},
		bson.M{"$sort": bson.M{"accuracy": 1}},
	}

	subjectCursor, err := config.GetCollection("mocktestsattempts").Aggregate(ctx, subjectPipeline)
	var subjectAccuracy []bson.M
	if err == nil {
		subjectCursor.All(ctx, &subjectAccuracy)
		subjectCursor.Close(ctx)
	}
	if subjectAccuracy == nil {
		subjectAccuracy = []bson.M{}
	}

	// ── 2. Mock test score trend (last 10 attempts) ───────────────────────────
	trendPipeline := bson.A{
		bson.M{"$match": bson.M{"userId": userID}},
		bson.M{"$sort": bson.M{"completedAt": -1}},
		bson.M{"$limit": 10},
		bson.M{
			"$lookup": bson.M{
				"from":         "mocktests",
				"localField":   "mockTestId",
				"foreignField": "_id",
				"as":           "test",
			},
		},
		bson.M{"$unwind": "$test"},
		bson.M{
			"$project": bson.M{
				"_id":         0,
				"score":       1,
				"totalMarks":  1,
				"timeTaken":   1,
				"completedAt": 1,
				"testTitle":   "$test.title",
				"testSubject": "$test.subject",
				"percent": bson.M{"$round": bson.A{
					bson.M{"$multiply": bson.A{bson.M{"$divide": bson.A{"$score", "$totalMarks"}}, 100}}, 1,
				}},
			},
		},
		bson.M{"$sort": bson.M{"completedAt": 1}},
	}

	trendCursor, err := config.GetCollection("mocktestsattempts").Aggregate(ctx, trendPipeline)
	var scoreTrend []bson.M
	if err == nil {
		trendCursor.All(ctx, &scoreTrend)
		trendCursor.Close(ctx)
	}
	if scoreTrend == nil {
		scoreTrend = []bson.M{}
	}

	// ── 3. Summary stats ──────────────────────────────────────────────────────
	totalAttempts, _ := config.GetCollection("mocktestsattempts").CountDocuments(ctx, bson.M{"userId": userID})

	sumPipeline := bson.A{
		bson.M{"$match": bson.M{"userId": userID}},
		bson.M{
			"$group": bson.M{
				"_id":        nil,
				"totalScore": bson.M{"$sum": "$score"},
				"totalMarks": bson.M{"$sum": "$totalMarks"},
				"bestScore":  bson.M{"$max": "$score"},
				"bestTotal":  bson.M{"$first": "$totalMarks"},
			},
		},
	}
	sumCursor, err := config.GetCollection("mocktestsattempts").Aggregate(ctx, sumPipeline)
	var sumResult []bson.M
	if err == nil {
		sumCursor.All(ctx, &sumResult)
		sumCursor.Close(ctx)
	}

	overallAccuracy := 0.0
	bestPercent := 0.0
	if len(sumResult) > 0 {
		s := sumResult[0]
		if total, ok := s["totalMarks"].(int32); ok && total > 0 {
			if score, ok2 := s["totalScore"].(int32); ok2 {
				overallAccuracy = float64(score) / float64(total) * 100
			}
		}
		if best, ok := s["bestScore"].(int32); ok {
			if total, ok2 := s["bestTotal"].(int32); ok2 && total > 0 {
				bestPercent = float64(best) / float64(total) * 100
			}
		}
	}

	// ── 4. Weak areas (subjects with accuracy < 60%) ──────────────────────────
	weakAreas := []string{}
	for _, s := range subjectAccuracy {
		if acc, ok := s["accuracy"].(float64); ok && acc < 60 {
			if sub, ok2 := s["subject"].(string); ok2 {
				weakAreas = append(weakAreas, sub)
			}
		}
	}

	// ── 5. Practice hub stats ──────────────────────────────────────────────────
	practiceCount, _ := config.GetCollection("userchapterprogress").CountDocuments(ctx, bson.M{"userId": userID})

	utils.Success(c, http.StatusOK, gin.H{
		"subjectAccuracy": subjectAccuracy,
		"scoreTrend":      scoreTrend,
		"weakAreas":       weakAreas,
		"summary": gin.H{
			"totalAttempts":   totalAttempts,
			"overallAccuracy": overallAccuracy,
			"bestPercent":     bestPercent,
			"chaptersAttempted": practiceCount,
		},
	}, "Success")
}
