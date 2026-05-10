package ws

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"navodaya-api/config"
	"navodaya-api/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ─── NewHub ───────────────────────────────────────────────────────────────────

func TestNewHub_Initialised(t *testing.T) {
	h := NewHub()
	if h == nil {
		t.Fatal("expected non-nil Hub")
	}
	if h.classes == nil {
		t.Error("expected non-nil classes map")
	}
	if h.Register == nil {
		t.Error("expected non-nil Register channel")
	}
	if h.Unregister == nil {
		t.Error("expected non-nil Unregister channel")
	}
}

func TestNewHub_EmptyClasses(t *testing.T) {
	h := NewHub()
	if h.ConnectedCount("any") != 0 {
		t.Error("expected 0 connected clients in empty hub")
	}
}

// ─── ConnectedCount ───────────────────────────────────────────────────────────

func TestConnectedCount_Empty(t *testing.T) {
	h := NewHub()
	if h.ConnectedCount("class1") != 0 {
		t.Error("expected 0 for unknown class")
	}
}

func TestConnectedCount_AfterRegister(t *testing.T) {
	h := NewHub()
	go h.Run()

	c := newTestClient(h, "class1", "u1", false)
	h.Register <- c
	time.Sleep(15 * time.Millisecond)

	if h.ConnectedCount("class1") != 1 {
		t.Errorf("expected 1 connected client, got %d", h.ConnectedCount("class1"))
	}
}

func TestConnectedCount_MultipleClasses(t *testing.T) {
	h := NewHub()
	go h.Run()

	c1 := newTestClient(h, "classA", "u1", false)
	c2 := newTestClient(h, "classB", "u2", false)
	h.Register <- c1
	h.Register <- c2
	time.Sleep(15 * time.Millisecond)

	if h.ConnectedCount("classA") != 1 {
		t.Errorf("classA: expected 1, got %d", h.ConnectedCount("classA"))
	}
	if h.ConnectedCount("classB") != 1 {
		t.Errorf("classB: expected 1, got %d", h.ConnectedCount("classB"))
	}
	if h.ConnectedCount("classC") != 0 {
		t.Errorf("classC: expected 0, got %d", h.ConnectedCount("classC"))
	}
}

func TestConnectedCount_MultipleClientsInClass(t *testing.T) {
	h := NewHub()
	go h.Run()

	for i := 0; i < 5; i++ {
		c := newTestClient(h, "class1", "user", false)
		h.Register <- c
	}
	time.Sleep(20 * time.Millisecond)

	if h.ConnectedCount("class1") != 5 {
		t.Errorf("expected 5 clients, got %d", h.ConnectedCount("class1"))
	}
}

// ─── Register / Unregister ────────────────────────────────────────────────────

func TestUnregister_RemovesClient(t *testing.T) {
	h := NewHub()
	go h.Run()

	c := newTestClient(h, "class1", "u1", false)
	h.Register <- c
	time.Sleep(15 * time.Millisecond)

	h.Unregister <- c
	time.Sleep(15 * time.Millisecond)

	if h.ConnectedCount("class1") != 0 {
		t.Errorf("expected 0 after unregister, got %d", h.ConnectedCount("class1"))
	}
}

func TestUnregister_CleanUpEmptyClass(t *testing.T) {
	h := NewHub()
	go h.Run()

	c := newTestClient(h, "class1", "u1", false)
	h.Register <- c
	time.Sleep(15 * time.Millisecond)

	h.Unregister <- c
	time.Sleep(15 * time.Millisecond)

	h.mu.RLock()
	_, exists := h.classes["class1"]
	h.mu.RUnlock()

	if exists {
		t.Error("expected empty class to be removed from classes map")
	}
}

func TestUnregister_NonExistent(t *testing.T) {
	h := NewHub()
	go h.Run()

	// Unregistering a client that was never registered should not panic
	c := newTestClient(h, "class1", "u1", false)
	h.Unregister <- c
	time.Sleep(15 * time.Millisecond)
	// No assertion needed — just check there's no panic
}

func TestUnregister_PartialRemoval(t *testing.T) {
	h := NewHub()
	go h.Run()

	c1 := newTestClient(h, "class1", "u1", false)
	c2 := newTestClient(h, "class1", "u2", false)
	h.Register <- c1
	h.Register <- c2
	time.Sleep(15 * time.Millisecond)

	h.Unregister <- c1
	time.Sleep(15 * time.Millisecond)

	if h.ConnectedCount("class1") != 1 {
		t.Errorf("expected 1 client after partial unregister, got %d", h.ConnectedCount("class1"))
	}
}

// ─── BroadcastToClass ─────────────────────────────────────────────────────────

func TestBroadcastToClass_AllClientsReceive(t *testing.T) {
	h := NewHub()
	go h.Run()

	c1 := newTestClient(h, "class1", "u1", false)
	c2 := newTestClient(h, "class1", "u2", true) // teacher also receives BroadcastToClass
	h.Register <- c1
	h.Register <- c2
	time.Sleep(15 * time.Millisecond)

	msg := Message{Type: EventClassEnd, Payload: nil}
	h.BroadcastToClass("class1", msg)

	for _, c := range []*Client{c1, c2} {
		select {
		case data := <-c.Send:
			var received Message
			if err := json.Unmarshal(data, &received); err != nil {
				t.Errorf("unmarshal error: %v", err)
			}
			if received.Type != EventClassEnd {
				t.Errorf("type: want %q, got %q", EventClassEnd, received.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("client did not receive broadcast message")
		}
	}
}

func TestBroadcastToClass_UnknownClass(t *testing.T) {
	h := NewHub()
	go h.Run()

	// Broadcasting to non-existent class should not panic
	msg := Message{Type: EventClassEnd, Payload: nil}
	h.BroadcastToClass("nonexistent-class", msg)
}

func TestBroadcastToClass_MessageContent(t *testing.T) {
	h := NewHub()
	go h.Run()

	c := newTestClient(h, "class1", "u1", false)
	h.Register <- c
	time.Sleep(15 * time.Millisecond)

	payload := ChatPayload{UserID: "u1", UserName: "Alice", Message: "hi", SentAt: "now"}
	msg := Message{Type: EventChatMessage, Payload: payload}
	h.BroadcastToClass("class1", msg)

	select {
	case data := <-c.Send:
		var received Message
		json.Unmarshal(data, &received)
		if received.Type != EventChatMessage {
			t.Errorf("expected chat_message, got %q", received.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("client did not receive message")
	}
}

// ─── BroadcastToStudents ──────────────────────────────────────────────────────

func TestBroadcastToStudents_SkipsTeachers(t *testing.T) {
	h := NewHub()
	go h.Run()

	student := newTestClient(h, "class1", "s1", false) // IsTeacher=false
	teacher := newTestClient(h, "class1", "t1", true)  // IsTeacher=true
	h.Register <- student
	h.Register <- teacher
	time.Sleep(15 * time.Millisecond)

	msg := Message{Type: EventQuizStart, Payload: QuizStartPayload{QuestionID: "q1"}}
	h.BroadcastToStudents("class1", msg)

	// Student should receive
	select {
	case <-student.Send:
		// expected
	case <-time.After(100 * time.Millisecond):
		t.Error("student did not receive BroadcastToStudents message")
	}

	// Teacher should NOT receive
	select {
	case <-teacher.Send:
		t.Error("teacher should not receive BroadcastToStudents message")
	case <-time.After(50 * time.Millisecond):
		// expected — timeout means no message was sent to teacher
	}
}

func TestBroadcastToStudents_MultipleStudents(t *testing.T) {
	h := NewHub()
	go h.Run()

	s1 := newTestClient(h, "class1", "s1", false)
	s2 := newTestClient(h, "class1", "s2", false)
	s3 := newTestClient(h, "class1", "s3", false)
	h.Register <- s1
	h.Register <- s2
	h.Register <- s3
	time.Sleep(20 * time.Millisecond)

	msg := Message{Type: EventQuizStart, Payload: nil}
	h.BroadcastToStudents("class1", msg)

	for _, s := range []*Client{s1, s2, s3} {
		select {
		case <-s.Send:
			// expected
		case <-time.After(100 * time.Millisecond):
			t.Error("a student did not receive broadcast message")
		}
	}
}

func TestBroadcastToStudents_UnknownClass(t *testing.T) {
	h := NewHub()
	// No panic expected
	h.BroadcastToStudents("unknown", Message{Type: EventQuizStart})
}

// ─── HandleClientMessage ──────────────────────────────────────────────────────

func TestHandleClientMessage_Chat(t *testing.T) {
	h := NewHub()
	go h.Run()

	sender := newTestClient(h, "class1", "u1", false)
	receiver := newTestClient(h, "class1", "u2", false)
	h.Register <- sender
	h.Register <- receiver
	time.Sleep(15 * time.Millisecond)

	msg := Message{
		Type: EventChatMessage,
		Payload: map[string]interface{}{
			"message": "Hello everyone",
		},
	}
	h.HandleClientMessage(sender, msg)

	// Both sender and receiver should get the broadcast
	received := 0
	timeout := time.After(100 * time.Millisecond)
	for received < 2 {
		select {
		case data := <-sender.Send:
			var m Message
			json.Unmarshal(data, &m)
			if m.Type != EventChatMessage {
				t.Errorf("expected chat_message, got %q", m.Type)
			}
			received++
		case data := <-receiver.Send:
			var m Message
			json.Unmarshal(data, &m)
			if m.Type != EventChatMessage {
				t.Errorf("expected chat_message, got %q", m.Type)
			}
			received++
		case <-timeout:
			t.Errorf("timeout: only received %d/2 messages", received)
			return
		}
	}
}

func TestHandleClientMessage_ChatSetsUserInfo(t *testing.T) {
	h := NewHub()
	go h.Run()

	c := newTestClient(h, "class1", "alice_id", false)
	c.UserName = "Alice"
	h.Register <- c
	time.Sleep(15 * time.Millisecond)

	h.HandleClientMessage(c, Message{
		Type:    EventChatMessage,
		Payload: map[string]interface{}{"message": "hi"},
	})

	select {
	case data := <-c.Send:
		var m Message
		json.Unmarshal(data, &m)
		// Payload should have userId and userName set from client
		payloadBytes, _ := json.Marshal(m.Payload)
		var p ChatPayload
		json.Unmarshal(payloadBytes, &p)
		if p.UserID != "alice_id" {
			t.Errorf("UserID: want alice_id, got %q", p.UserID)
		}
		if p.UserName != "Alice" {
			t.Errorf("UserName: want Alice, got %q", p.UserName)
		}
		if p.SentAt == "" {
			t.Error("expected SentAt to be set")
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("did not receive chat broadcast")
	}
}

func TestHandleClientMessage_QuizAnswer_TeacherIgnored(t *testing.T) {
	h := NewHub()
	go h.Run()

	teacher := newTestClient(h, "class1", "t1", true)
	h.Register <- teacher
	time.Sleep(15 * time.Millisecond)

	// Teachers cannot submit quiz answers — message should be silently ignored
	h.HandleClientMessage(teacher, Message{
		Type: EventQuizAnswer,
		Payload: map[string]interface{}{
			"questionId":    "507f1f77bcf86cd799439011",
			"selectedIndex": 2,
			"timeTaken":     10,
		},
	})

	select {
	case <-teacher.Send:
		t.Error("teacher should not receive any message back for quiz answer attempt")
	case <-time.After(50 * time.Millisecond):
		// expected — no response to teacher quiz answer
	}
}

func TestHandleClientMessage_QuizAnswer_InvalidPayload(t *testing.T) {
	h := NewHub()
	go h.Run()

	student := newTestClient(h, "class1", "s1", false)
	h.Register <- student
	time.Sleep(15 * time.Millisecond)

	// Malformed payload — should not panic
	h.HandleClientMessage(student, Message{
		Type:    EventQuizAnswer,
		Payload: "invalid-not-an-object",
	})
}

func TestHandleClientMessage_UnknownType(t *testing.T) {
	h := NewHub()
	go h.Run()

	c := newTestClient(h, "class1", "u1", false)
	h.Register <- c
	time.Sleep(15 * time.Millisecond)

	// Unknown event type — should be silently ignored, no panic
	h.HandleClientMessage(c, Message{Type: "unknown_type", Payload: nil})

	select {
	case <-c.Send:
		// Some implementations may echo unknown types — either is fine
	case <-time.After(30 * time.Millisecond):
		// expected — unknown types are ignored
	}
}

func TestHandleClientMessage_ChatInvalidPayload(t *testing.T) {
	h := NewHub()
	go h.Run()

	c := newTestClient(h, "class1", "u1", false)
	h.Register <- c
	time.Sleep(15 * time.Millisecond)

	// Payload that can't be marshalled to ChatPayload — should not panic
	h.HandleClientMessage(c, Message{
		Type:    EventChatMessage,
		Payload: func() {}, // not JSON-serialisable
	})
}

// ─── helpers ─────────────────────────────────────────────────────────────────

func newTestClient(h *Hub, classID, userID string, isTeacher bool) *Client {
	return &Client{
		Hub:       h,
		Send:      make(chan []byte, 256),
		ClassID:   classID,
		UserID:    userID,
		UserName:  userID,
		IsTeacher: isTeacher,
	}
}

// ─── GetLeaderboard ───────────────────────────────────────────────────────────

func TestGetLeaderboard_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "quizanswers")

	h := NewHub()
	questionID := primitive.NewObjectID()
	entries := h.GetLeaderboard(questionID)

	if entries == nil {
		t.Fatal("expected non-nil entries")
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for empty collection, got %d", len(entries))
	}
}

func TestGetLeaderboard_WithAnswers(t *testing.T) {
	requireDB(t)
	dropCollection(t, "quizanswers")

	questionID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	ctx := context.Background()

	config.GetCollection("quizanswers").InsertOne(ctx, models.QuizAnswer{
		ID:               primitive.NewObjectID(),
		LiveClassID:      classID,
		QuestionID:       questionID,
		UserID:           primitive.NewObjectID(),
		UserName:         "Alice",
		SelectedIndex:    0,
		IsCorrect:        true,
		TimeTakenSeconds: 5,
		SubmittedAt:      time.Now(),
	})
	config.GetCollection("quizanswers").InsertOne(ctx, models.QuizAnswer{
		ID:               primitive.NewObjectID(),
		LiveClassID:      classID,
		QuestionID:       questionID,
		UserID:           primitive.NewObjectID(),
		UserName:         "Bob",
		SelectedIndex:    1,
		IsCorrect:        false,
		TimeTakenSeconds: 10,
		SubmittedAt:      time.Now(),
	})

	h := NewHub()
	entries := h.GetLeaderboard(questionID)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	// Correct answers ranked first
	if !entries[0].IsCorrect {
		t.Error("expected first entry to be correct answer")
	}
	if entries[0].UserName != "Alice" {
		t.Errorf("first entry: want Alice, got %s", entries[0].UserName)
	}
	if entries[0].Rank != 1 {
		t.Errorf("first entry rank: want 1, got %d", entries[0].Rank)
	}
}

func TestGetLeaderboard_CorrectSortedBeforeWrong(t *testing.T) {
	requireDB(t)
	dropCollection(t, "quizanswers")

	questionID := primitive.NewObjectID()
	classID := primitive.NewObjectID()
	ctx := context.Background()

	// Wrong answer inserted first
	config.GetCollection("quizanswers").InsertOne(ctx, models.QuizAnswer{
		ID:               primitive.NewObjectID(),
		LiveClassID:      classID,
		QuestionID:       questionID,
		UserID:           primitive.NewObjectID(),
		UserName:         "Wrong",
		IsCorrect:        false,
		TimeTakenSeconds: 1,
		SubmittedAt:      time.Now(),
	})
	// Correct answer inserted second
	config.GetCollection("quizanswers").InsertOne(ctx, models.QuizAnswer{
		ID:               primitive.NewObjectID(),
		LiveClassID:      classID,
		QuestionID:       questionID,
		UserID:           primitive.NewObjectID(),
		UserName:         "Right",
		IsCorrect:        true,
		TimeTakenSeconds: 8,
		SubmittedAt:      time.Now(),
	})

	h := NewHub()
	entries := h.GetLeaderboard(questionID)

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].UserName != "Right" {
		t.Errorf("correct answer should be ranked first, got %s", entries[0].UserName)
	}
}

// ─── persistQuizAnswer ────────────────────────────────────────────────────────

func TestPersistQuizAnswer_InvalidIDs(t *testing.T) {
	// persistQuizAnswer returns early (silently) on invalid hex IDs — no panic
	h := NewHub()
	go h.Run()

	client := newTestClient(h, "not-valid-class", "not-valid-user", false)

	payload := QuizAnswerPayload{
		QuestionID:    "not-valid-question",
		SelectedIndex: 0,
		TimeTaken:     5,
	}
	// Should not panic — invalid IDs cause early return
	h.persistQuizAnswer(client, payload)
}

func TestPersistQuizAnswer_QuestionNotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "livequestions")

	h := NewHub()
	go h.Run()

	classID := primitive.NewObjectID()
	userID := primitive.NewObjectID()
	questionID := primitive.NewObjectID()

	client := newTestClient(h, classID.Hex(), userID.Hex(), false)
	payload := QuizAnswerPayload{
		QuestionID:    questionID.Hex(), // doesn't exist in DB
		SelectedIndex: 0,
		TimeTaken:     5,
	}
	// Should not panic — missing question causes early return
	h.persistQuizAnswer(client, payload)
}

func TestPersistQuizAnswer_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "livequestions")
	dropCollection(t, "quizanswers")

	classID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	// Insert a live question in the DB so persistQuizAnswer can look it up
	question := models.LiveQuestion{
		ID:           primitive.NewObjectID(),
		LiveClassID:  classID,
		Text:         "Test?",
		Options:      []string{"A", "B"},
		CorrectIndex: 0,
		IsActive:     true,
		TimerSeconds: 30,
		CreatedAt:    time.Now(),
	}
	config.GetCollection("livequestions").InsertOne(context.Background(), question)

	h := NewHub()
	go h.Run()

	client := newTestClient(h, classID.Hex(), userID.Hex(), false)
	client.UserName = "TestUser"

	payload := QuizAnswerPayload{
		QuestionID:    question.ID.Hex(),
		SelectedIndex: 0, // matches CorrectIndex → isCorrect=true
		TimeTaken:     5,
	}
	h.persistQuizAnswer(client, payload)

	// Give the goroutine a moment to write to DB
	time.Sleep(50 * time.Millisecond)

	count, err := config.GetCollection("quizanswers").CountDocuments(
		context.Background(), map[string]interface{}{"questionId": question.ID},
	)
	if err != nil {
		t.Fatalf("count error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 quiz answer saved, got %d", count)
	}
}
