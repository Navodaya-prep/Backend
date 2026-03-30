package ws

// EventType defines all real-time event types exchanged over WebSocket.
type EventType string

const (
	EventChatMessage EventType = "chat_message"
	EventQuizStart   EventType = "quiz_start"
	EventQuizEnd     EventType = "quiz_end"
	EventQuizAnswer  EventType = "quiz_answer"
	EventClassEnd    EventType = "class_end"
	EventError       EventType = "error"
)

// Message is the envelope for all WebSocket payloads.
type Message struct {
	Type    EventType   `json:"type"`
	Payload interface{} `json:"payload"`
}

// ChatPayload is sent and received for chat messages.
type ChatPayload struct {
	UserID   string `json:"userId"`
	UserName string `json:"userName"`
	Message  string `json:"message"`
	SentAt   string `json:"sentAt"`
}

// QuizStartPayload is broadcast to students when a teacher pushes a question.
type QuizStartPayload struct {
	QuestionID   string   `json:"questionId"`
	Text         string   `json:"text"`
	Options      []string `json:"options"`
	TimerSeconds int      `json:"timerSeconds"`
	IsReadOnly   bool     `json:"isReadOnly"`
}

// QuizEndPayload is broadcast when a question timer expires or teacher ends it.
type QuizEndPayload struct {
	QuestionID   string             `json:"questionId"`
	CorrectIndex int                `json:"correctIndex"`
	Leaderboard  []LeaderboardEntry `json:"leaderboard"`
}

// LeaderboardEntry is a single row in the quiz leaderboard.
type LeaderboardEntry struct {
	Rank      int    `json:"rank"`
	UserID    string `json:"userId"`
	UserName  string `json:"userName"`
	IsCorrect bool   `json:"isCorrect"`
	TimeTaken int    `json:"timeTaken"` // seconds
}

// QuizAnswerPayload is sent by a student when they submit an answer.
type QuizAnswerPayload struct {
	QuestionID    string `json:"questionId"`
	SelectedIndex int    `json:"selectedIndex"`
	TimeTaken     int    `json:"timeTaken"` // seconds elapsed since quiz_start
}
