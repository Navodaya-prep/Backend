package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/navodayasarthi/api/config"
	"github.com/navodayasarthi/api/models"
	"github.com/navodayasarthi/api/utils"
)

const premiumPlan = "premium_lifetime"

// premiumPricePaise is the one-time premium price in paise (₹499 by default,
// overridable via PREMIUM_PRICE_PAISE).
func premiumPricePaise() int {
	if v := os.Getenv("PREMIUM_PRICE_PAISE"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return 49900
}

// grantPremium marks a user premium and flips the matching payment to "paid".
// paymentFilter selects the payment record to update (by linkId or orderId).
// Idempotent: safe to call from the status check, verify, and the webhook.
func grantPremium(ctx context.Context, userID primitive.ObjectID, paymentFilter bson.M, paymentID string) {
	config.GetCollection("users").UpdateOne(ctx,
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"isPremium": true, "updatedAt": time.Now()}})

	if paymentFilter == nil {
		return
	}
	set := bson.M{"status": "paid", "updatedAt": time.Now()}
	if paymentID != "" {
		set["paymentId"] = paymentID
	}
	config.GetCollection("payments").UpdateOne(ctx, paymentFilter, bson.M{"$set": set})
}

// CreatePremiumPaymentLink — POST /payments/create-link
// Creates a Razorpay payment link for the one-time premium unlock.
func CreatePremiumPaymentLink(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var user models.User
	if err := config.GetCollection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}
	if user.IsPremium {
		utils.Success(c, http.StatusOK, gin.H{"alreadyPremium": true}, "Already premium")
		return
	}
	if !utils.RazorpayConfigured() {
		utils.ErrorRes(c, http.StatusServiceUnavailable, "PAYMENTS_UNAVAILABLE", "Payments are not configured yet")
		return
	}

	amount := premiumPricePaise()
	paymentID := primitive.NewObjectID()

	link, err := utils.CreatePaymentLink(
		amount,
		"NavodayaSarthi Premium (lifetime)",
		paymentID.Hex(), // reference_id — lets us map the link back to this record
		user.Name,
		user.Phone,
		map[string]string{"userId": userID.Hex(), "plan": premiumPlan},
	)
	if err != nil {
		utils.ErrorRes(c, http.StatusBadGateway, "PAYMENT_INIT_FAILED", "Could not start payment. Please try again.")
		return
	}

	_, _ = config.GetCollection("payments").InsertOne(ctx, models.Payment{
		ID:          paymentID,
		UserID:      userID,
		Provider:    "razorpay",
		LinkID:      link.ID,
		AmountPaise: amount,
		Plan:        premiumPlan,
		Status:      "created",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	utils.Success(c, http.StatusOK, gin.H{
		"linkId":      link.ID,
		"linkUrl":     link.ShortURL,
		"amountPaise": amount,
	}, "Payment link created")
}

// CreatePremiumOrder — POST /payments/create-order
// Creates a Razorpay order for the native checkout (UPI intent) flow and
// returns everything the mobile SDK needs to open checkout.
func CreatePremiumOrder(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var user models.User
	if err := config.GetCollection("users").FindOne(ctx, bson.M{"_id": userID}).Decode(&user); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "User not found")
		return
	}
	if user.IsPremium {
		utils.Success(c, http.StatusOK, gin.H{"alreadyPremium": true}, "Already premium")
		return
	}
	if !utils.RazorpayConfigured() {
		utils.ErrorRes(c, http.StatusServiceUnavailable, "PAYMENTS_UNAVAILABLE", "Payments are not configured yet")
		return
	}

	amount := premiumPricePaise()
	paymentID := primitive.NewObjectID()

	order, err := utils.CreateOrder(amount, paymentID.Hex(), map[string]string{
		"userId": userID.Hex(),
		"plan":   premiumPlan,
	})
	if err != nil {
		utils.ErrorRes(c, http.StatusBadGateway, "PAYMENT_INIT_FAILED", "Could not start payment. Please try again.")
		return
	}

	_, _ = config.GetCollection("payments").InsertOne(ctx, models.Payment{
		ID:          paymentID,
		UserID:      userID,
		Provider:    "razorpay",
		OrderID:     order.ID,
		AmountPaise: amount,
		Plan:        premiumPlan,
		Status:      "created",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	})

	utils.Success(c, http.StatusOK, gin.H{
		"orderId":     order.ID,
		"amount":      amount,
		"currency":    "INR",
		"keyId":       utils.RazorpayKeyID(),
		"name":        user.Name,
		"contact":     user.Phone,
		"description": "NavodayaSarthi Premium (lifetime)",
	}, "Order created")
}

// VerifyPremiumPayment — POST /payments/verify
// Body: { razorpay_order_id, razorpay_payment_id, razorpay_signature }
// Verifies the checkout signature server-side and grants premium on success.
func VerifyPremiumPayment(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))

	var body struct {
		OrderID   string `json:"razorpay_order_id" binding:"required"`
		PaymentID string `json:"razorpay_payment_id" binding:"required"`
		Signature string `json:"razorpay_signature" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		utils.ErrorRes(c, http.StatusBadRequest, "MISSING_FIELDS", "Payment fields are required")
		return
	}

	if !utils.VerifyPaymentSignature(body.OrderID, body.PaymentID, body.Signature) {
		utils.ErrorRes(c, http.StatusBadRequest, "VERIFICATION_FAILED", "Payment could not be verified")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Only grant against an order we created for this user.
	var payment models.Payment
	if err := config.GetCollection("payments").FindOne(ctx,
		bson.M{"orderId": body.OrderID, "userId": userID}).Decode(&payment); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Order not found")
		return
	}

	grantPremium(ctx, userID, bson.M{"orderId": body.OrderID}, body.PaymentID)

	utils.Success(c, http.StatusOK, gin.H{"isPremium": true}, "Payment verified")
}

// GetPaymentStatus — GET /payments/status/:linkId
// Queries Razorpay for the link's status; grants premium on first "paid".
// Used by the app when the checkout browser closes (instant unlock, while the
// webhook acts as the reliable backstop).
func GetPaymentStatus(c *gin.Context) {
	userIDStr, _ := c.Get("userId")
	userID, _ := primitive.ObjectIDFromHex(userIDStr.(string))
	linkID := c.Param("linkId")

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	var payment models.Payment
	if err := config.GetCollection("payments").FindOne(ctx,
		bson.M{"linkId": linkID, "userId": userID}).Decode(&payment); err != nil {
		utils.ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "Payment not found")
		return
	}

	status := payment.Status
	if status != "paid" && utils.RazorpayConfigured() {
		if link, err := utils.FetchPaymentLink(linkID); err == nil {
			status = link.Status
			if link.Status == "paid" {
				grantPremium(ctx, userID, bson.M{"linkId": linkID}, "")
				status = "paid"
			}
		}
	}

	utils.Success(c, http.StatusOK, gin.H{
		"status":    status,
		"isPremium": status == "paid",
	}, "Success")
}

// RazorpayWebhook — POST /payments/webhook  (public, signature-verified)
// The reliable source of truth: Razorpay calls this when a link is paid.
func RazorpayWebhook(c *gin.Context) {
	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot read body"})
		return
	}

	if !utils.VerifyRazorpayWebhook(rawBody, c.GetHeader("X-Razorpay-Signature")) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid signature"})
		return
	}

	var event struct {
		Event   string `json:"event"`
		Payload struct {
			PaymentLink struct {
				Entity struct {
					ID          string            `json:"id"`
					Status      string            `json:"status"`
					ReferenceID string            `json:"reference_id"`
					Notes       map[string]string `json:"notes"`
				} `json:"entity"`
			} `json:"payment_link"`
			Order struct {
				Entity struct {
					ID    string            `json:"id"`
					Notes map[string]string `json:"notes"`
				} `json:"entity"`
			} `json:"order"`
			Payment struct {
				Entity struct {
					ID      string            `json:"id"`
					OrderID string            `json:"order_id"`
					Notes   map[string]string `json:"notes"`
				} `json:"entity"`
			} `json:"payment"`
		} `json:"payload"`
	}
	if err := json.Unmarshal(rawBody, &event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "bad payload"})
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// resolveUser maps a set of candidate notes / a payment lookup to a user id.
	resolveUser := func(notes map[string]string, fallbackFilter bson.M) primitive.ObjectID {
		if uid, ok := notes["userId"]; ok {
			if id, err := primitive.ObjectIDFromHex(uid); err == nil {
				return id
			}
		}
		if fallbackFilter != nil {
			var p models.Payment
			if err := config.GetCollection("payments").FindOne(ctx, fallbackFilter).Decode(&p); err == nil {
				return p.UserID
			}
		}
		return primitive.ObjectID{}
	}

	switch event.Event {
	case "payment_link.paid": // payment-link flow (legacy / fallback)
		link := event.Payload.PaymentLink.Entity
		var fallback bson.M
		if link.ReferenceID != "" {
			if refID, err := primitive.ObjectIDFromHex(link.ReferenceID); err == nil {
				fallback = bson.M{"_id": refID}
			}
		}
		if userID := resolveUser(link.Notes, fallback); !userID.IsZero() {
			grantPremium(ctx, userID, bson.M{"linkId": link.ID}, event.Payload.Payment.Entity.ID)
		}

	case "order.paid", "payment.captured": // native checkout (UPI intent) flow
		orderID := event.Payload.Order.Entity.ID
		if orderID == "" {
			orderID = event.Payload.Payment.Entity.OrderID
		}
		notes := event.Payload.Order.Entity.Notes
		if len(notes) == 0 {
			notes = event.Payload.Payment.Entity.Notes
		}
		if userID := resolveUser(notes, bson.M{"orderId": orderID}); !userID.IsZero() && orderID != "" {
			grantPremium(ctx, userID, bson.M{"orderId": orderID}, event.Payload.Payment.Entity.ID)
		}

	default:
		// Acknowledge unrelated events without action.
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
