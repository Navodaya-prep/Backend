package ws

import (
	"encoding/json"
	"testing"
	"time"
)

// ─── EventType constants ──────────────────────────────────────────────────────

func TestEventTypeValues(t *testing.T) {
	cases := []struct {
		name string
		got  EventType
		want EventType
	}{
		{"EventChatMessage", EventChatMessage, "chat_message"},
		{"EventQuizStart", EventQuizStart, "quiz_start"},
		{"EventQuizEnd", EventQuizEnd, "quiz_end"},
		{"EventQuizAnswer", EventQuizAnswer, "quiz_answer"},
		{"EventClassEnd", EventClassEnd, "class_end"},
		{"EventError", EventError, "error"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("want %q, got %q", tc.want, tc.got)
			}
		})
	}
}

func TestEventTypeDistinct(t *testing.T) {
	seen := make(map[EventType]bool)
	all := []EventType{EventChatMessage, EventQuizStart, EventQuizEnd, EventQuizAnswer, EventClassEnd, EventError}
	for _, e := range all {
		if seen[e] {
			t.Errorf("duplicate EventType value: %q", e)
		}
		seen[e] = true
	}
}

// ─── Message serialization ────────────────────────────────────────────────────

func TestMessage_MarshalChat(t *testing.T) {
	msg := Message{
		Type: EventChatMessage,
		Payload: ChatPayload{
			UserID:   "u1",
			UserName: "Alice",
			Message:  "Hello!",
			SentAt:   time.Now().UTC().Format(time.RFC3339),
		},
	}

	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if out["type"] != string(EventChatMessage) {
		t.Errorf("type: want %q, got %v", EventChatMessage, out["type"])
	}
	if out["payload"] == nil {
		t.Error("expected non-nil payload")
	}
}

func TestMessage_Unmarshal(t *testing.T) {
	raw := `{"type":"chat_message","payload":{"userId":"u2","userName":"Bob","message":"Hi","sentAt":"2024-01-01T00:00:00Z"}}`

	var msg Message
	if err := json.Unmarshal([]byte(raw), &msg); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if msg.Type != EventChatMessage {
		t.Errorf("type: want %q, got %q", EventChatMessage, msg.Type)
	}
	if msg.Payload == nil {
		t.Error("expected non-nil payload")
	}
}

func TestMessage_UnknownType(t *testing.T) {
	msg := Message{Type: "unknown_event", Payload: nil}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var out map[string]interface{}
	json.Unmarshal(data, &out)
	if out["type"] != "unknown_event" {
		t.Errorf("expected unknown_event, got %v", out["type"])
	}
}

func TestMessage_NilPayload(t *testing.T) {
	msg := Message{Type: EventClassEnd, Payload: nil}
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty marshalled message")
	}
}

// ─── ChatPayload ──────────────────────────────────────────────────────────────

func TestChatPayload_JSONRoundTrip(t *testing.T) {
	p := ChatPayload{
		UserID:   "user123",
		UserName: "Alice",
		Message:  "Hello, world!",
		SentAt:   "2024-01-01T12:00:00Z",
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var out ChatPayload
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if out.UserID != p.UserID {
		t.Errorf("UserID: want %q, got %q", p.UserID, out.UserID)
	}
	if out.UserName != p.UserName {
		t.Errorf("UserName: want %q, got %q", p.UserName, out.UserName)
	}
	if out.Message != p.Message {
		t.Errorf("Message: want %q, got %q", p.Message, out.Message)
	}
	if out.SentAt != p.SentAt {
		t.Errorf("SentAt: want %q, got %q", p.SentAt, out.SentAt)
	}
}

// ─── QuizStartPayload ────────────────────────────────────────────────────────

func TestQuizStartPayload_JSONRoundTrip(t *testing.T) {
	p := QuizStartPayload{
		QuestionID:   "q1",
		Text:         "What is 2+2?",
		Options:      []string{"1", "2", "3", "4"},
		TimerSeconds: 30,
		IsReadOnly:   false,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var out QuizStartPayload
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if out.QuestionID != p.QuestionID {
		t.Errorf("QuestionID: want %q, got %q", p.QuestionID, out.QuestionID)
	}
	if len(out.Options) != len(p.Options) {
		t.Errorf("Options length: want %d, got %d", len(p.Options), len(out.Options))
	}
	if out.TimerSeconds != p.TimerSeconds {
		t.Errorf("TimerSeconds: want %d, got %d", p.TimerSeconds, out.TimerSeconds)
	}
	if out.IsReadOnly != p.IsReadOnly {
		t.Errorf("IsReadOnly: want %v, got %v", p.IsReadOnly, out.IsReadOnly)
	}
}

func TestQuizStartPayload_ReadOnly(t *testing.T) {
	p := QuizStartPayload{IsReadOnly: true, TimerSeconds: 0}
	data, _ := json.Marshal(p)
	var out QuizStartPayload
	json.Unmarshal(data, &out)
	if !out.IsReadOnly {
		t.Error("expected IsReadOnly=true")
	}
}

// ─── QuizEndPayload ───────────────────────────────────────────────────────────

func TestQuizEndPayload_JSONRoundTrip(t *testing.T) {
	p := QuizEndPayload{
		QuestionID:   "q2",
		CorrectIndex: 3,
		Leaderboard: []LeaderboardEntry{
			{Rank: 1, UserID: "u1", UserName: "Alice", IsCorrect: true, TimeTaken: 5},
			{Rank: 2, UserID: "u2", UserName: "Bob", IsCorrect: false, TimeTaken: 10},
		},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var out QuizEndPayload
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if out.CorrectIndex != p.CorrectIndex {
		t.Errorf("CorrectIndex: want %d, got %d", p.CorrectIndex, out.CorrectIndex)
	}
	if len(out.Leaderboard) != 2 {
		t.Errorf("Leaderboard length: want 2, got %d", len(out.Leaderboard))
	}
	if out.Leaderboard[0].IsCorrect != true {
		t.Error("first entry should be correct")
	}
}

// ─── QuizAnswerPayload ────────────────────────────────────────────────────────

func TestQuizAnswerPayload_JSONRoundTrip(t *testing.T) {
	p := QuizAnswerPayload{
		QuestionID:    "q3",
		SelectedIndex: 2,
		TimeTaken:     15,
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var out QuizAnswerPayload
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if out.QuestionID != p.QuestionID {
		t.Errorf("QuestionID: want %q, got %q", p.QuestionID, out.QuestionID)
	}
	if out.SelectedIndex != p.SelectedIndex {
		t.Errorf("SelectedIndex: want %d, got %d", p.SelectedIndex, out.SelectedIndex)
	}
	if out.TimeTaken != p.TimeTaken {
		t.Errorf("TimeTaken: want %d, got %d", p.TimeTaken, out.TimeTaken)
	}
}

// ─── LeaderboardEntry ─────────────────────────────────────────────────────────

func TestLeaderboardEntry_JSONRoundTrip(t *testing.T) {
	e := LeaderboardEntry{
		Rank:      1,
		UserID:    "user1",
		UserName:  "Alice",
		IsCorrect: true,
		TimeTaken: 7,
	}

	data, err := json.Marshal(e)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var out LeaderboardEntry
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if out.Rank != e.Rank {
		t.Errorf("Rank: want %d, got %d", e.Rank, out.Rank)
	}
	if out.UserID != e.UserID {
		t.Errorf("UserID: want %q, got %q", e.UserID, out.UserID)
	}
	if out.IsCorrect != e.IsCorrect {
		t.Errorf("IsCorrect: want %v, got %v", e.IsCorrect, out.IsCorrect)
	}
}
