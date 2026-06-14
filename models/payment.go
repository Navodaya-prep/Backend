package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Payment records a premium-purchase attempt and its outcome.
// Status: "created" → "paid" (or stays "created" if abandoned).
type Payment struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID      primitive.ObjectID `bson:"userId" json:"userId"`
	Provider    string             `bson:"provider" json:"provider"`                   // "razorpay"
	LinkID      string             `bson:"linkId,omitempty" json:"linkId,omitempty"`   // payment link id (link flow)
	OrderID     string             `bson:"orderId,omitempty" json:"orderId,omitempty"` // order id (native checkout flow)
	PaymentID   string             `bson:"paymentId,omitempty" json:"paymentId,omitempty"`
	AmountPaise int                `bson:"amountPaise" json:"amountPaise"`
	Plan        string             `bson:"plan" json:"plan"` // "premium_lifetime"
	Status      string             `bson:"status" json:"status"`
	CreatedAt   time.Time          `bson:"createdAt" json:"createdAt"`
	UpdatedAt   time.Time          `bson:"updatedAt" json:"updatedAt"`
}
