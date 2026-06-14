package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/navodayasarthi/api/config"
	"github.com/navodayasarthi/api/models"
	"github.com/navodayasarthi/api/utils"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// ─── Admin: Subject CRUD ──────────────────────────────────────────────────────

// chapterQCount holds question tallies for a single chapter.
type chapterQCount struct {
	Total   int
	Premium int
}

// countQuestionsByChapter returns chapterID → {total, premium} question counts.
// Pass nil chapterIDs to count across all chapters.
func countQuestionsByChapter(ctx context.Context, chapterIDs []primitive.ObjectID) map[primitive.ObjectID]chapterQCount {
	match := bson.M{}
	if chapterIDs != nil {
		match = bson.M{"chapterId": bson.M{"$in": chapterIDs}}
	}
	pipeline := []bson.M{
		{"$match": match},
		{"$group": bson.M{
			"_id":     "$chapterId",
			"total":   bson.M{"$sum": 1},
			"premium": bson.M{"$sum": bson.M{"$cond": bson.A{"$isPremium", 1, 0}}},
		}},
	}
	res := map[primitive.ObjectID]chapterQCount{}
	cursor, err := config.GetCollection("questions").Aggregate(ctx, pipeline)
	if err != nil {
		return res
	}
	defer cursor.Close(ctx)
	var rows []struct {
		ID      primitive.ObjectID `bson:"_id"`
		Total   int                `bson:"total"`
		Premium int                `bson:"premium"`
	}
	cursor.All(ctx, &rows)
	for _, r := range rows {
		res[r.ID] = chapterQCount{Total: r.Total, Premium: r.Premium}
	}
	return res
}

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

	// Map each chapter to its subject, then roll up question counts per subject.
	chCursor, _ := config.GetCollection("chapters").Find(ctx,
		bson.M{"subjectId": bson.M{"$ne": nil}})
	var chapters []models.Chapter
	if chCursor != nil {
		chCursor.All(ctx, &chapters)
		chCursor.Close(ctx)
	}
	chapterToSubject := make(map[primitive.ObjectID]primitive.ObjectID, len(chapters))
	for _, ch := range chapters {
		if ch.SubjectID != nil {
			chapterToSubject[ch.ID] = *ch.SubjectID
		}
	}

	counts := countQuestionsByChapter(ctx, nil)
	subjTotals := make(map[primitive.ObjectID]chapterQCount)
	for chID, cc := range counts {
		if sid, ok := chapterToSubject[chID]; ok {
			t := subjTotals[sid]
			t.Total += cc.Total
			t.Premium += cc.Premium
			subjTotals[sid] = t
		}
	}

	type subjectWithCounts struct {
		models.Subject
		QuestionCount int `json:"questionCount"`
		PremiumCount  int `json:"premiumCount"`
		FreeCount     int `json:"freeCount"`
	}
	result := make([]subjectWithCounts, len(subjects))
	for i, s := range subjects {
		cc := subjTotals[s.ID]
		result[i] = subjectWithCounts{
			Subject:       s,
			QuestionCount: cc.Total,
			PremiumCount:  cc.Premium,
			FreeCount:     cc.Total - cc.Premium,
		}
	}

	utils.Success(c, http.StatusOK, gin.H{"subjects": result}, "Success")
}

// AdminCreateSubject — POST /admin/practice/subjects
func AdminCreateSubject(c *gin.Context) {
	var body struct {
		Name        string `json:"name" binding:"required"`
		NameHi      string `json:"nameHi"`
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
		NameHi:      body.NameHi,
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
		NameHi      string `json:"nameHi"`
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
		"name": body.Name, "nameHi": body.NameHi, "icon": body.Icon, "color": body.Color,
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

	ids := make([]primitive.ObjectID, len(chapters))
	for i, ch := range chapters {
		ids[i] = ch.ID
	}
	counts := countQuestionsByChapter(ctx, ids)

	type chapterWithCounts struct {
		models.Chapter
		QuestionCount int `json:"questionCount"`
		PremiumCount  int `json:"premiumCount"`
		FreeCount     int `json:"freeCount"`
	}
	result := make([]chapterWithCounts, len(chapters))
	for i, ch := range chapters {
		cc := counts[ch.ID]
		result[i] = chapterWithCounts{
			Chapter:       ch,
			QuestionCount: cc.Total,
			PremiumCount:  cc.Premium,
			FreeCount:     cc.Total - cc.Premium,
		}
	}

	utils.Success(c, http.StatusOK, gin.H{"chapters": result}, "Success")
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
		TitleHi     string `json:"titleHi"`
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
		TitleHi:     body.TitleHi,
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
		TitleHi     string `json:"titleHi"`
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
		"title": body.Title, "titleHi": body.TitleHi, "description": body.Description,
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
		Text         string                  `json:"text" binding:"required"`
		TextHi       string                  `json:"textHi"`
		ImageURL     string                  `json:"imageUrl"`
		Options      []models.QuestionOption `json:"options" binding:"required"`
		CorrectIndex int                     `json:"correctIndex"`
		Explanation  string                  `json:"explanation"`
		Difficulty   string                  `json:"difficulty"` // easy | medium | hard
		ClassLevel   string                  `json:"classLevel"`
		Tags         []string                `json:"tags"`
		IsPremium    bool                    `json:"isPremium"`
		IsPYQ        bool                    `json:"isPYQ"`    // Previous Year Question
		ExamYear     string                  `json:"examYear"` // e.g., "2024", "2023"
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", err.Error())
		return
	}
	if len(body.Options) < 2 {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_OPTIONS", "At least 2 options required")
		return
	}
	// Validate option types
	for _, opt := range body.Options {
		if opt.Type != "text" && opt.Type != "image" {
			utils.ErrorRes(c, http.StatusBadRequest, "INVALID_OPTIONS", "Option type must be 'text' or 'image'")
			return
		}
		if opt.Value == "" {
			utils.ErrorRes(c, http.StatusBadRequest, "INVALID_OPTIONS", "Option value cannot be empty")
			return
		}
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
		TextHi:       body.TextHi,
		ImageURL:     body.ImageURL,
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
		Text         *string                 `json:"text"`
		TextHi       *string                 `json:"textHi"`
		ImageURL     *string                 `json:"imageUrl"`
		Options      []models.QuestionOption `json:"options"`
		CorrectIndex *int                    `json:"correctIndex"`
		Explanation  *string                 `json:"explanation"`
		Difficulty   *string                 `json:"difficulty"`
		ClassLevel   *string                 `json:"classLevel"`
		Tags         []string                `json:"tags"`
		IsPremium    *bool                   `json:"isPremium"`
		IsPYQ        *bool                   `json:"isPYQ"`
		ExamYear     *string                 `json:"examYear"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "INVALID_BODY", err.Error())
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Only set fields that were actually present in the request payload.
	set := bson.M{}
	if body.Text != nil         { set["text"] = *body.Text }
	if body.TextHi != nil       { set["textHi"] = *body.TextHi }
	if body.ImageURL != nil     { set["imageUrl"] = *body.ImageURL }
	if body.Options != nil      { set["options"] = body.Options }
	if body.CorrectIndex != nil { set["correctIndex"] = *body.CorrectIndex }
	if body.Explanation != nil  { set["explanation"] = *body.Explanation }
	if body.Difficulty != nil   { set["difficulty"] = *body.Difficulty }
	if body.ClassLevel != nil   { set["classLevel"] = *body.ClassLevel }
	if body.Tags != nil         { set["tags"] = body.Tags }
	if body.IsPremium != nil    { set["isPremium"] = *body.IsPremium }
	if body.IsPYQ != nil        { set["isPYQ"] = *body.IsPYQ }
	if body.ExamYear != nil     { set["examYear"] = *body.ExamYear }

	if len(set) == 0 {
		utils.ErrorRes(c, http.StatusBadRequest, "EMPTY_UPDATE", "No fields to update")
		return
	}

	update := bson.M{"$set": set}
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

	// Attach chapter count and total question count per subject
	type SubjectWithCount struct {
		models.Subject
		ChapterCount  int `json:"chapterCount"`
		QuestionCount int `json:"questionCount"`
	}
	result := make([]SubjectWithCount, len(subjects))
	for i, s := range subjects {
		chapterCount, _ := config.GetCollection("chapters").CountDocuments(ctx, bson.M{"subjectId": s.ID})

		// Collect chapter IDs then count questions across them
		chapterCursor, _ := config.GetCollection("chapters").Find(ctx,
			bson.M{"subjectId": s.ID}, options.Find().SetProjection(bson.M{"_id": 1}))
		var chapterDocs []struct{ ID primitive.ObjectID `bson:"_id"` }
		chapterCursor.All(ctx, &chapterDocs)
		chapterCursor.Close(ctx)

		chapterIDs := make([]primitive.ObjectID, len(chapterDocs))
		for j, ch := range chapterDocs {
			chapterIDs[j] = ch.ID
		}
		questionCount := 0
		if len(chapterIDs) > 0 {
			qCount, _ := config.GetCollection("questions").CountDocuments(ctx, bson.M{"chapterId": bson.M{"$in": chapterIDs}})
			questionCount = int(qCount)
		}

		result[i] = SubjectWithCount{Subject: s, ChapterCount: int(chapterCount), QuestionCount: questionCount}
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

// GetChapterQuestions — GET /practice/chapters/:id/questions?pyq=true
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

	filter := bson.M{"chapterId": chapterID}
	if c.Query("pyq") == "true" {
		filter["isPYQ"] = true
	}

	cursor, err := config.GetCollection("questions").Find(ctx,
		filter,
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

	// Hide answer data for premium questions from non-premium users.
	redactPremiumQuestions(questions, userIsPremium(userID))

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

	// Non-premium users cannot submit premium questions — drop them so they
	// are neither scored nor marked solved (gate can't be bypassed via POST).
	if !userIsPremium(userID) {
		filtered := questions[:0]
		for _, q := range questions {
			if !q.IsPremium {
				filtered = append(filtered, q)
			}
		}
		questions = filtered
	}

	// Score and build detailed result
	type DetailItem struct {
		QuestionID  string `json:"questionId"`
		Text        string `json:"text"`
		TextHi      string `json:"textHi,omitempty"`
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
			TextHi:      q.TextHi,
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
