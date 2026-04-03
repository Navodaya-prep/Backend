package handlers

import (
	"context"
	"net/http"
	"time"

	"navodaya-api/config"
	"navodaya-api/models"
	"navodaya-api/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ─── Admin: Subject CRUD ──────────────────────────────────────────────────────

// AdminListSubjects — GET /admin/practice/subjects
func AdminListSubjects(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("subjects").Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "order", Value: 1}}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch subjects")
		return
	}
	defer cursor.Close(ctx)

	var subjects []models.Subject
	cursor.All(ctx, &subjects)
	if subjects == nil {
		subjects = []models.Subject{}
	}
	utils.Success(c, http.StatusOK, gin.H{"subjects": subjects}, "Success")
}

// AdminCreateSubject — POST /admin/practice/subjects
func AdminCreateSubject(c *gin.Context) {
	var body struct {
		Name        string `json:"name" binding:"required"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
		Description string `json:"description"`
		Order       int    `json:"order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", err.Error())
		return
	}

	subject := models.Subject{
		ID:          primitive.NewObjectID(),
		Name:        body.Name,
		Icon:        body.Icon,
		Color:       body.Color,
		Description: body.Description,
		Order:       body.Order,
		CreatedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := config.GetCollection("subjects").InsertOne(ctx, subject); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create subject")
		return
	}
	utils.Success(c, http.StatusCreated, gin.H{"subject": subject}, "Subject created")
}

// AdminUpdateSubject — PUT /admin/practice/subjects/:id
func AdminUpdateSubject(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid subject ID")
		return
	}

	var body struct {
		Name        string `json:"name"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
		Description string `json:"description"`
		Order       int    `json:"order"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{
		"name": body.Name, "icon": body.Icon, "color": body.Color,
		"description": body.Description, "order": body.Order,
	}}
	res, err := config.GetCollection("subjects").UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil || res.MatchedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Subject not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Subject updated")
}

// AdminDeleteSubject — DELETE /admin/practice/subjects/:id
func AdminDeleteSubject(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid subject ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("subjects").DeleteOne(ctx, bson.M{"_id": id})
	if err != nil || res.DeletedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Subject not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Subject deleted")
}

// ─── Admin: Chapter CRUD ──────────────────────────────────────────────────────

// AdminListChapters — GET /admin/practice/subjects/:id/chapters
func AdminListChapters(c *gin.Context) {
	subjectID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid subject ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("chapters").Find(ctx,
		bson.M{"subjectId": subjectID},
		options.Find().SetSort(bson.D{{Key: "order", Value: 1}}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch chapters")
		return
	}
	defer cursor.Close(ctx)

	var chapters []models.Chapter
	cursor.All(ctx, &chapters)
	if chapters == nil {
		chapters = []models.Chapter{}
	}
	utils.Success(c, http.StatusOK, gin.H{"chapters": chapters}, "Success")
}

// AdminCreateChapter — POST /admin/practice/subjects/:id/chapters
func AdminCreateChapter(c *gin.Context) {
	subjectID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid subject ID")
		return
	}

	var body struct {
		Title       string `json:"title" binding:"required"`
		Description string `json:"description"`
		Order       int    `json:"order"`
		IsPremium   bool   `json:"isPremium"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", err.Error())
		return
	}

	chapter := models.Chapter{
		ID:          primitive.NewObjectID(),
		SubjectID:   &subjectID,
		Title:       body.Title,
		Description: body.Description,
		Order:       body.Order,
		IsPremium:   body.IsPremium,
		CreatedAt:   time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := config.GetCollection("chapters").InsertOne(ctx, chapter); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create chapter")
		return
	}
	utils.Success(c, http.StatusCreated, gin.H{"chapter": chapter}, "Chapter created")
}

// AdminUpdateChapter — PUT /admin/practice/chapters/:id
func AdminUpdateChapter(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid chapter ID")
		return
	}

	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		Order       int    `json:"order"`
		IsPremium   bool   `json:"isPremium"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{
		"title": body.Title, "description": body.Description,
		"order": body.Order, "isPremium": body.IsPremium,
	}}
	res, err := config.GetCollection("chapters").UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil || res.MatchedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Chapter not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Chapter updated")
}

// AdminDeleteChapter — DELETE /admin/practice/chapters/:id
func AdminDeleteChapter(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid chapter ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("chapters").DeleteOne(ctx, bson.M{"_id": id})
	if err != nil || res.DeletedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Chapter not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Chapter deleted")
}

// ─── Admin: Question CRUD ─────────────────────────────────────────────────────

// AdminListChapterQuestions — GET /admin/practice/chapters/:id/questions
func AdminListChapterQuestions(c *gin.Context) {
	chapterID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid chapter ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to find questions associated with this chapter
	filter := bson.M{"chapterId": chapterID}
	cursor, err := config.GetCollection("questions").Find(ctx, filter,
		options.Find().SetSort(bson.D{{Key: "difficulty", Value: 1}, {Key: "createdAt", Value: 1}}))
	if err != nil {
		// Log the error for debugging
		println("Error finding questions:", err.Error())
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch questions: "+err.Error())
		return
	}
	defer cursor.Close(ctx)

	var questions []models.Question
	if err := cursor.All(ctx, &questions); err != nil {
		println("Error decoding questions:", err.Error())
		utils.ErrorRes(c, http.StatusInternalServerError, "DECODE_FAILED", "Failed to decode questions")
		return
	}

	if questions == nil {
		questions = []models.Question{}
	}
	utils.Success(c, http.StatusOK, gin.H{"questions": questions}, "Success")
}

// AdminCreateQuestion — POST /admin/practice/chapters/:id/questions
func AdminCreateQuestion(c *gin.Context) {
	chapterID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid chapter ID")
		return
	}

	var body struct {
		Text         string   `json:"text" binding:"required"`
		Options      []string `json:"options" binding:"required"`
		CorrectIndex int      `json:"correctIndex"`
		Explanation  string   `json:"explanation"`
		Difficulty   string   `json:"difficulty"` // easy | medium | hard
		ClassLevel   string   `json:"classLevel"`
		Tags         []string `json:"tags"`
		IsPremium    bool     `json:"isPremium"`
		IsPYQ        bool     `json:"isPYQ"`    // Previous Year Question
		ExamYear     string   `json:"examYear"` // e.g., "2024", "2023"
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", err.Error())
		return
	}
	if len(body.Options) < 2 {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_OPTIONS", "At least 2 options required")
		return
	}
	if body.Difficulty == "" {
		body.Difficulty = "medium"
	}
	if body.Tags == nil {
		body.Tags = []string{}
	}

	question := models.Question{
		ID:           primitive.NewObjectID(),
		ChapterID:    &chapterID,
		Text:         body.Text,
		Options:      body.Options,
		CorrectIndex: body.CorrectIndex,
		Explanation:  body.Explanation,
		Difficulty:   body.Difficulty,
		ClassLevel:   body.ClassLevel,
		Tags:         body.Tags,
		IsPremium:    body.IsPremium,
		IsPYQ:        body.IsPYQ,
		ExamYear:     body.ExamYear,
		CreatedAt:    time.Now(),
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if _, err := config.GetCollection("questions").InsertOne(ctx, question); err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "CREATE_FAILED", "Failed to create question")
		return
	}
	utils.Success(c, http.StatusCreated, gin.H{"question": question}, "Question created")
}

// AdminUpdateQuestion — PUT /admin/practice/questions/:id
func AdminUpdateQuestion(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid question ID")
		return
	}

	var body struct {
		Text         string   `json:"text"`
		Options      []string `json:"options"`
		CorrectIndex int      `json:"correctIndex"`
		Explanation  string   `json:"explanation"`
		Difficulty   string   `json:"difficulty"`
		ClassLevel   string   `json:"classLevel"`
		Tags         []string `json:"tags"`
		IsPremium    bool     `json:"isPremium"`
		IsPYQ        bool     `json:"isPYQ"`
		ExamYear     string   `json:"examYear"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	update := bson.M{"$set": bson.M{
		"text": body.Text, "options": body.Options, "correctIndex": body.CorrectIndex,
		"explanation": body.Explanation, "difficulty": body.Difficulty,
		"classLevel": body.ClassLevel, "tags": body.Tags, "isPremium": body.IsPremium,
		"isPYQ": body.IsPYQ, "examYear": body.ExamYear,
	}}
	res, err := config.GetCollection("questions").UpdateOne(ctx, bson.M{"_id": id}, update)
	if err != nil || res.MatchedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Question not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Question updated")
}

// AdminDeleteQuestion — DELETE /admin/practice/questions/:id
func AdminDeleteQuestion(c *gin.Context) {
	id, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid question ID")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := config.GetCollection("questions").DeleteOne(ctx, bson.M{"_id": id})
	if err != nil || res.DeletedCount == 0 {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Question not found")
		return
	}
	utils.Success(c, http.StatusOK, nil, "Question deleted")
}

// ─── Student: Practice Hub ────────────────────────────────────────────────────

// ListSubjects — GET /practice/subjects
func ListSubjects(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("subjects").Find(ctx, bson.M{},
		options.Find().SetSort(bson.D{{Key: "order", Value: 1}}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch subjects")
		return
	}
	defer cursor.Close(ctx)

	var subjects []models.Subject
	cursor.All(ctx, &subjects)
	if subjects == nil {
		subjects = []models.Subject{}
	}

	// Attach chapter count per subject
	type SubjectWithCount struct {
		models.Subject
		ChapterCount int `json:"chapterCount"`
	}
	result := make([]SubjectWithCount, len(subjects))
	for i, s := range subjects {
		count, _ := config.GetCollection("chapters").CountDocuments(ctx, bson.M{"subjectId": s.ID})
		result[i] = SubjectWithCount{Subject: s, ChapterCount: int(count)}
	}
	utils.Success(c, http.StatusOK, gin.H{"subjects": result}, "Success")
}

// ListSubjectChapters — GET /practice/subjects/:id/chapters
func ListSubjectChapters(c *gin.Context) {
	subjectID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid subject ID")
		return
	}
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("chapters").Find(ctx,
		bson.M{"subjectId": subjectID},
		options.Find().SetSort(bson.D{{Key: "order", Value: 1}}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch chapters")
		return
	}
	defer cursor.Close(ctx)

	var chapters []models.Chapter
	cursor.All(ctx, &chapters)
	if chapters == nil {
		chapters = []models.Chapter{}
	}

	// Attach total question count and solved count for each chapter
	type ChapterWithProgress struct {
		models.Chapter
		QuestionCount int `json:"questionCount"`
		SolvedCount   int `json:"solvedCount"`
	}
	result := make([]ChapterWithProgress, len(chapters))
	for i, ch := range chapters {
		total, _ := config.GetCollection("questions").CountDocuments(ctx, bson.M{"chapterId": ch.ID})

		var progress models.UserChapterProgress
		var solved int
		if err := config.GetCollection("userchapterprogress").FindOne(ctx,
			bson.M{"userId": userID, "chapterId": ch.ID}).Decode(&progress); err == nil {
			solved = len(progress.SolvedQuestionIDs)
		}
		result[i] = ChapterWithProgress{Chapter: ch, QuestionCount: int(total), SolvedCount: solved}
	}
	utils.Success(c, http.StatusOK, gin.H{"chapters": result}, "Success")
}

// GetChapterQuestions — GET /practice/chapters/:id/questions
// Returns all questions + the user's solved question IDs for that chapter
func GetChapterQuestions(c *gin.Context) {
	chapterID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid chapter ID")
		return
	}
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("questions").Find(ctx,
		bson.M{"chapterId": chapterID},
		options.Find().SetSort(bson.D{{Key: "difficulty", Value: 1}, {Key: "createdAt", Value: 1}}))
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch questions")
		return
	}
	defer cursor.Close(ctx)

	var questions []models.Question
	cursor.All(ctx, &questions)
	if questions == nil {
		questions = []models.Question{}
	}

	// Fetch user's solved IDs for this chapter
	var progress models.UserChapterProgress
	solvedIDs := []string{}
	if err := config.GetCollection("userchapterprogress").FindOne(ctx,
		bson.M{"userId": userID, "chapterId": chapterID}).Decode(&progress); err == nil {
		for _, id := range progress.SolvedQuestionIDs {
			solvedIDs = append(solvedIDs, id.Hex())
		}
	}

	utils.Success(c, http.StatusOK, gin.H{
		"questions": questions,
		"solvedIds": solvedIDs,
	}, "Success")
}

// SubmitChapterPractice — POST /practice/chapters/:id/submit
// Body: { answers: { "<questionId>": selectedIndex, ... } }
func SubmitChapterPractice(c *gin.Context) {
	chapterID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_ID", "Invalid chapter ID")
		return
	}
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	var body struct {
		Answers map[string]int `json:"answers" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "answers map is required")
		return
	}

	// Fetch the submitted questions
	questionIDs := make([]primitive.ObjectID, 0, len(body.Answers))
	for idStr := range body.Answers {
		qid, err := primitive.ObjectIDFromHex(idStr)
		if err == nil {
			questionIDs = append(questionIDs, qid)
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cursor, err := config.GetCollection("questions").Find(ctx,
		bson.M{"_id": bson.M{"$in": questionIDs}})
	if err != nil {
		utils.ErrorRes(c, http.StatusInternalServerError, "FETCH_FAILED", "Failed to fetch questions")
		return
	}
	defer cursor.Close(ctx)

	var questions []models.Question
	cursor.All(ctx, &questions)

	// Score and build detailed result
	type DetailItem struct {
		QuestionID  string `json:"questionId"`
		Text        string `json:"text"`
		SelectedIdx int    `json:"selectedIndex"`
		CorrectIdx  int    `json:"correctIndex"`
		IsCorrect   bool   `json:"isCorrect"`
		Difficulty  string `json:"difficulty"`
		Explanation string `json:"explanation"`
	}

	correct := 0
	detailed := make([]DetailItem, 0, len(questions))
	solvedIDs := make([]primitive.ObjectID, 0)

	for _, q := range questions {
		selected, exists := body.Answers[q.ID.Hex()]
		if !exists {
			selected = -1
		}
		isCorrect := exists && selected == q.CorrectIndex
		if isCorrect {
			correct++
		}
		solvedIDs = append(solvedIDs, q.ID)
		detailed = append(detailed, DetailItem{
			QuestionID:  q.ID.Hex(),
			Text:        q.Text,
			SelectedIdx: selected,
			CorrectIdx:  q.CorrectIndex,
			IsCorrect:   isCorrect,
			Difficulty:  q.Difficulty,
			Explanation: q.Explanation,
		})
	}

	total := len(questions)
	percent := 0
	if total > 0 {
		percent = (correct * 100) / total
	}

	// Upsert progress — add solved question IDs (deduplicated via $addToSet)
	if len(solvedIDs) > 0 {
		filter := bson.M{"userId": userID, "chapterId": chapterID}
		update := bson.M{
			"$addToSet": bson.M{"solvedQuestionIds": bson.M{"$each": solvedIDs}},
			"$set":      bson.M{"updatedAt": time.Now()},
			"$setOnInsert": bson.M{
				"_id": primitive.NewObjectID(),
			},
		}
		opts := options.Update().SetUpsert(true)
		config.GetCollection("userchapterprogress").UpdateOne(ctx, filter, update, opts)
	}

	utils.Success(c, http.StatusOK, gin.H{
		"result": gin.H{
			"correct":  correct,
			"total":    total,
			"percent":  percent,
			"detailed": detailed,
		},
	}, "Practice submitted")
}
