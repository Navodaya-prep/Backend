package models

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsontype"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// QuestionOption represents a single answer option which can be text or image.
// Type: "text" (default) or "image"
// Value: the text content or the image URL
type QuestionOption struct {
	Type  string `bson:"type" json:"type"`   // "text" or "image"
	Value string `bson:"value" json:"value"` // text content or image URL
}

// UnmarshalBSONValue handles both legacy string options and current {type,value} object options.
func (o *QuestionOption) UnmarshalBSONValue(t bsontype.Type, data []byte) error {
	switch t {
	case bsontype.String:
		str, ok := bson.RawValue{Type: t, Value: data}.StringValueOK()
		if !ok {
			return fmt.Errorf("failed to decode string option")
		}
		o.Type = "text"
		o.Value = str
		return nil
	case bsontype.EmbeddedDocument:
		var doc struct {
			Type  string `bson:"type"`
			Value string `bson:"value"`
		}
		if err := bson.UnmarshalValue(t, data, &doc); err != nil {
			return err
		}
		o.Type = doc.Type
		o.Value = doc.Value
		return nil
	default:
		return fmt.Errorf("unsupported BSON type for QuestionOption: %s", t)
	}
}

type Question struct {
	ID           primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	ChapterID    *primitive.ObjectID `bson:"chapterId,omitempty" json:"chapterId,omitempty"`
	Subject      string              `bson:"subject" json:"subject"`
	Text         string              `bson:"text" json:"text"`
	ImageURL     string              `bson:"imageUrl,omitempty" json:"imageUrl,omitempty"` // optional image for question
	Options      []QuestionOption    `bson:"options" json:"options"`
	CorrectIndex int                 `bson:"correctIndex" json:"correctIndex"`
	Explanation  string              `bson:"explanation" json:"explanation"`
	Difficulty   string              `bson:"difficulty" json:"difficulty"`
	ClassLevel   string              `bson:"classLevel" json:"classLevel"`
	IsPremium    bool                `bson:"isPremium" json:"isPremium"`
	IsPYQ        bool                `bson:"isPYQ" json:"isPYQ"`       // Previous Year Question
	ExamYear     string              `bson:"examYear" json:"examYear"` // e.g., "2024", "2023"
	Tags         []string            `bson:"tags" json:"tags"`
	CreatedAt    time.Time           `bson:"createdAt" json:"createdAt"`
}
