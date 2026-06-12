package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// NotificationTemplate is an admin-editable message for one notification type.
// Title and Body support {placeholder} substitution at send time.
type NotificationTemplate struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Key          string             `bson:"key" json:"key"`   // stable identifier, e.g. "daily_challenge"
	Name         string             `bson:"name" json:"name"` // human-friendly label shown in admin panel
	Description  string             `bson:"description" json:"description"`
	Kind         string             `bson:"kind" json:"kind"` // "scheduled" | "event"
	Title        string             `bson:"title" json:"title"`
	Body         string             `bson:"body" json:"body"`
	Enabled      bool               `bson:"enabled" json:"enabled"`
	Placeholders []string           `bson:"placeholders" json:"placeholders"`           // variables available in title/body
	SendTime     string             `bson:"sendTime,omitempty" json:"sendTime"`         // "HH:MM" IST, scheduled kind only
	LastSentDate string             `bson:"lastSentDate,omitempty" json:"lastSentDate"` // "YYYY-MM-DD" dedupe for scheduled sends
	UpdatedAt    time.Time          `bson:"updatedAt" json:"updatedAt"`
	UpdatedBy    string             `bson:"updatedBy,omitempty" json:"updatedBy"`
}

// NotificationLog records every push that went out, for the admin panel history.
type NotificationLog struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Key        string             `bson:"key" json:"key"` // template key or "broadcast"
	Title      string             `bson:"title" json:"title"`
	Body       string             `bson:"body" json:"body"`
	Recipients int                `bson:"recipients" json:"recipients"`
	SentBy     string             `bson:"sentBy,omitempty" json:"sentBy"` // admin email for manual broadcasts, "system" otherwise
	CreatedAt  time.Time          `bson:"createdAt" json:"createdAt"`
}
