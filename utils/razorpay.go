package utils

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const razorpayBaseURL = "https://api.razorpay.com/v1"

// razorpayCreds returns the configured key id / secret, or ok=false if payments
// are not configured (so callers can fail gracefully in dev).
func razorpayCreds() (keyID, keySecret string, ok bool) {
	keyID = os.Getenv("RAZORPAY_KEY_ID")
	keySecret = os.Getenv("RAZORPAY_KEY_SECRET")
	return keyID, keySecret, keyID != "" && keySecret != ""
}

// RazorpayConfigured reports whether Razorpay keys are present.
func RazorpayConfigured() bool {
	_, _, ok := razorpayCreds()
	return ok
}

// RazorpayKeyID returns the public key id (safe to send to the client).
func RazorpayKeyID() string {
	return os.Getenv("RAZORPAY_KEY_ID")
}

// Order is the subset of Razorpay's order entity we care about.
type Order struct {
	ID       string `json:"id"`     // order_...
	Amount   int    `json:"amount"` // paise
	Currency string `json:"currency"`
	Status   string `json:"status"` // created | attempted | paid
	Receipt  string `json:"receipt"`
}

// CreateOrder creates a Razorpay order for the native checkout (UPI intent)
// flow. notes are echoed back in the webhook so we can identify the user.
func CreateOrder(amountPaise int, receipt string, notes map[string]string) (*Order, error) {
	keyID, keySecret, ok := razorpayCreds()
	if !ok {
		return nil, fmt.Errorf("razorpay not configured")
	}

	body := map[string]interface{}{
		"amount":   amountPaise,
		"currency": "INR",
		"receipt":  receipt,
		"notes":    notes,
	}
	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, razorpayBaseURL+"/orders", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(keyID, keySecret)

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("razorpay create order failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var order Order
	if err := json.Unmarshal(respBody, &order); err != nil {
		return nil, err
	}
	return &order, nil
}

// VerifyPaymentSignature validates the signature returned by the checkout SDK
// after a successful payment: HMAC-SHA256(order_id|payment_id, key_secret).
func VerifyPaymentSignature(orderID, paymentID, signature string) bool {
	_, keySecret, ok := razorpayCreds()
	if !ok || orderID == "" || paymentID == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(keySecret))
	mac.Write([]byte(orderID + "|" + paymentID))
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// PaymentLink is the subset of Razorpay's payment-link entity we care about.
type PaymentLink struct {
	ID          string `json:"id"`
	ShortURL    string `json:"short_url"`
	Status      string `json:"status"` // created | paid | cancelled | expired
	ReferenceID string `json:"reference_id"`
	Amount      int    `json:"amount"`
}

// CreatePaymentLink creates a Razorpay payment link and returns it. notes are
// stored on the link and echoed back in the webhook so we can identify the user.
func CreatePaymentLink(amountPaise int, description, referenceID, customerName, customerContact string, notes map[string]string) (*PaymentLink, error) {
	keyID, keySecret, ok := razorpayCreds()
	if !ok {
		return nil, fmt.Errorf("razorpay not configured")
	}

	body := map[string]interface{}{
		"amount":          amountPaise,
		"currency":        "INR",
		"accept_partial":  false,
		"description":     description,
		"reference_id":    referenceID,
		"reminder_enable": false,
		"notify":          map[string]bool{"sms": false, "email": false},
		"notes":           notes,
	}
	if customerName != "" || customerContact != "" {
		body["customer"] = map[string]string{"name": customerName, "contact": customerContact}
	}

	payload, _ := json.Marshal(body)
	req, err := http.NewRequest(http.MethodPost, razorpayBaseURL+"/payment_links", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(keyID, keySecret)

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("razorpay create link failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var link PaymentLink
	if err := json.Unmarshal(respBody, &link); err != nil {
		return nil, err
	}
	return &link, nil
}

// FetchPaymentLink retrieves the current status of a payment link by id.
func FetchPaymentLink(linkID string) (*PaymentLink, error) {
	keyID, keySecret, ok := razorpayCreds()
	if !ok {
		return nil, fmt.Errorf("razorpay not configured")
	}

	req, err := http.NewRequest(http.MethodGet, razorpayBaseURL+"/payment_links/"+linkID, nil)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(keyID, keySecret)

	resp, err := (&http.Client{Timeout: 15 * time.Second}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("razorpay fetch link failed (%d): %s", resp.StatusCode, string(respBody))
	}

	var link PaymentLink
	if err := json.Unmarshal(respBody, &link); err != nil {
		return nil, err
	}
	return &link, nil
}

// VerifyRazorpayWebhook checks the X-Razorpay-Signature header against the raw
// request body using the configured webhook secret (HMAC-SHA256).
func VerifyRazorpayWebhook(rawBody []byte, signature string) bool {
	secret := os.Getenv("RAZORPAY_WEBHOOK_SECRET")
	if secret == "" || signature == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(rawBody)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}
