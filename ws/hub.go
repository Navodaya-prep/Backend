package ws

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
	"navodaya-api/config"
	"navodaya-api/models"
)

// Hub manages all active WebSocket connections grouped by live class ID.
type Hub struct {
	classes    map[string]map[*Client]bool
	mu         sync.RWMutex
	Register   chan *Client
	Unregister chan *Client
}

// GlobalHub is the singleton hub started at server boot.
var GlobalHub = NewHub()

func NewHub() *Hub {
	return &Hub{
		classes:    make(map[string]map[*Client]bool),
		Register:   make(chan *Client, 256),
		Unregister: make(chan *Client, 256),
	}
}

// Run processes register/unregister events. Must be called in a goroutine.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.Register:
			h.mu.Lock()
			if h.classes[client.ClassID] == nil {
				h.classes[client.ClassID] = make(map[*Client]bool)
			}
			h.classes[client.ClassID][client] = true
			h.mu.Unlock()
			log.Printf("[ws] client joined class=%s user=%s teacher=%v", client.ClassID, client.UserName, client.IsTeacher)

		case client := <-h.Unregister:
			h.mu.Lock()
			if clients, ok := h.classes[client.ClassID]; ok {
				if _, ok := clients[client]; ok {
					delete(clients, client)
					close(client.Send)
					if len(clients) == 0 {
						delete(h.classes, client.ClassID)
					}
				}
			}
			h.mu.Unlock()
			log.Printf("[ws] client left class=%s user=%s", client.ClassID, client.UserName)
		}
	}
}

// BroadcastToClass sends a message to every client in a class.
func (h *Hub) BroadcastToClass(classID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.classes[classID] {
		select {
		case c.Send <- data:
		default:
			close(c.Send)
			delete(h.classes[classID], c)
		}
	}
}

// BroadcastToStudents sends a message only to non-teacher clients in a class.
func (h *Hub) BroadcastToStudents(classID string, msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.classes[classID] {
		if !c.IsTeacher {
			select {
			case c.Send <- data:
			default:
				close(c.Send)
				delete(h.classes[classID], c)
			}
		}
	}
}

// ConnectedCount returns how many clients are in a class.
func (h *Hub) ConnectedCount(classID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.classes[classID])
}

// HandleClientMessage processes inbound messages from a client.
func (h *Hub) HandleClientMessage(c *Client, msg Message) {
	switch msg.Type {

	case EventChatMessage:
		payloadBytes, _ := json.Marshal(msg.Payload)
		var p ChatPayload
		if err := json.Unmarshal(payloadBytes, &p); err != nil {
			return
		}
		p.UserID = c.UserID
		p.UserName = c.UserName
		p.SentAt = time.Now().UTC().Format(time.RFC3339)
		h.BroadcastToClass(c.ClassID, Message{Type: EventChatMessage, Payload: p})

	case EventQuizAnswer:
		if c.IsTeacher {
			return // teachers cannot submit answers
		}
		payloadBytes, _ := json.Marshal(msg.Payload)
		var p QuizAnswerPayload
		if err := json.Unmarshal(payloadBytes, &p); err != nil {
			return
		}
		go h.persistQuizAnswer(c, p)
	}
}

// persistQuizAnswer saves a student's answer and upserts to avoid duplicates.
func (h *Hub) persistQuizAnswer(c *Client, p QuizAnswerPayload) {
	questionID, err := primitive.ObjectIDFromHex(p.QuestionID)
	if err != nil {
		return
	}
	classID, err := primitive.ObjectIDFromHex(c.ClassID)
	if err != nil {
		return
	}
	userID, err := primitive.ObjectIDFromHex(c.UserID)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var question models.LiveQuestion
	if err := config.GetCollection("livequestions").FindOne(ctx, bson.M{"_id": questionID}).Decode(&question); err != nil {
		return
	}

	isCorrect := p.SelectedIndex == question.CorrectIndex
	now := time.Now()

	answer := models.QuizAnswer{
		LiveClassID:      classID,
		QuestionID:       questionID,
		UserID:           userID,
		UserName:         c.UserName,
		SelectedIndex:    p.SelectedIndex,
		IsCorrect:        isCorrect,
		TimeTakenSeconds: p.TimeTaken,
		SubmittedAt:      now,
	}

	filter := bson.M{"questionId": questionID, "userId": userID}
	update := bson.M{"$set": answer, "$setOnInsert": bson.M{"_id": primitive.NewObjectID()}}
	opts := options.Update().SetUpsert(true)

	if _, err := config.GetCollection("quizanswers").UpdateOne(ctx, filter, update, opts); err != nil {
		log.Printf("[ws] failed to save quiz answer: %v", err)
	}
}

// GetLeaderboard fetches ranked answers for a question: correct first, then by time taken.
func (h *Hub) GetLeaderboard(questionID primitive.ObjectID) []LeaderboardEntry {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	findOpts := options.Find().SetSort(bson.D{
		{Key: "isCorrect", Value: -1},
		{Key: "timeTakenSeconds", Value: 1},
	})

	cursor, err := config.GetCollection("quizanswers").Find(ctx, bson.M{"questionId": questionID}, findOpts)
	if err != nil {
		return []LeaderboardEntry{}
	}
	defer cursor.Close(ctx)

	var answers []models.QuizAnswer
	cursor.All(ctx, &answers)

	entries := make([]LeaderboardEntry, len(answers))
	for i, a := range answers {
		entries[i] = LeaderboardEntry{
			Rank:      i + 1,
			UserID:    a.UserID.Hex(),
			UserName:  a.UserName,
			IsCorrect: a.IsCorrect,
			TimeTaken: a.TimeTakenSeconds,
		}
	}
	return entries
}
