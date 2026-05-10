package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"navodaya-api/config"
	"navodaya-api/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// ─── SendOTP ──────────────────────────────────────────────────────────────────

func TestSendOTP_MissingPhone(t *testing.T) {
	r := newRouter("POST", "/auth/send-otp", SendOTP)
	w := doRequest(r, "POST", "/auth/send-otp", map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestSendOTP_EmptyPhone(t *testing.T) {
	r := newRouter("POST", "/auth/send-otp", SendOTP)
	w := doRequest(r, "POST", "/auth/send-otp", map[string]interface{}{"phone": ""}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for empty phone, got %d", w.Code)
	}
}

func TestSendOTP_InvalidPhone_TooShort(t *testing.T) {
	r := newRouter("POST", "/auth/send-otp", SendOTP)
	w := doRequest(r, "POST", "/auth/send-otp", map[string]interface{}{"phone": "98765"}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for too-short phone, got %d", w.Code)
	}
}

func TestSendOTP_InvalidPhone_StartsWithFive(t *testing.T) {
	// Indian numbers must start with 6-9
	r := newRouter("POST", "/auth/send-otp", SendOTP)
	w := doRequest(r, "POST", "/auth/send-otp", map[string]interface{}{"phone": "5876543210"}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for phone starting with 5, got %d", w.Code)
	}
}

func TestSendOTP_InvalidPhone_Letters(t *testing.T) {
	r := newRouter("POST", "/auth/send-otp", SendOTP)
	w := doRequest(r, "POST", "/auth/send-otp", map[string]interface{}{"phone": "98765ABCDE"}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for phone with letters, got %d", w.Code)
	}
}

func TestSendOTP_InvalidPhone_TooLong(t *testing.T) {
	r := newRouter("POST", "/auth/send-otp", SendOTP)
	w := doRequest(r, "POST", "/auth/send-otp", map[string]interface{}{"phone": "98765432101"}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for 11-digit phone, got %d", w.Code)
	}
}

func TestSendOTP_ValidPhone_DevMode(t *testing.T) {
	requireDB(t)
	dropCollection(t, "otps")

	// OTP_DEV_MODE=true skips SMS — no external call needed
	r := newRouter("POST", "/auth/send-otp", SendOTP)
	w := doRequest(r, "POST", "/auth/send-otp", map[string]interface{}{"phone": "9876543210"}, nil)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

func TestSendOTP_ValidPhone_StartsWith6(t *testing.T) {
	requireDB(t)
	dropCollection(t, "otps")

	r := newRouter("POST", "/auth/send-otp", SendOTP)
	w := doRequest(r, "POST", "/auth/send-otp", map[string]interface{}{"phone": "6123456789"}, nil)
	if w.Code != http.StatusOK {
		t.Errorf("want 200 for phone starting with 6, got %d", w.Code)
	}
}

func TestSendOTP_ValidPhone_StartsWith7(t *testing.T) {
	requireDB(t)
	dropCollection(t, "otps")

	r := newRouter("POST", "/auth/send-otp", SendOTP)
	w := doRequest(r, "POST", "/auth/send-otp", map[string]interface{}{"phone": "7123456789"}, nil)
	if w.Code != http.StatusOK {
		t.Errorf("want 200 for phone starting with 7, got %d", w.Code)
	}
}

// ─── VerifyOTP ────────────────────────────────────────────────────────────────

func TestVerifyOTP_MissingPhone(t *testing.T) {
	r := newRouter("POST", "/auth/verify-otp", VerifyOTP)
	w := doRequest(r, "POST", "/auth/verify-otp", map[string]interface{}{"otp": "123456"}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestVerifyOTP_MissingOTP(t *testing.T) {
	r := newRouter("POST", "/auth/verify-otp", VerifyOTP)
	w := doRequest(r, "POST", "/auth/verify-otp", map[string]interface{}{"phone": "9876543210"}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestVerifyOTP_MissingBothFields(t *testing.T) {
	r := newRouter("POST", "/auth/verify-otp", VerifyOTP)
	w := doRequest(r, "POST", "/auth/verify-otp", map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestVerifyOTP_InvalidOTP(t *testing.T) {
	requireDB(t)
	dropCollection(t, "otps")

	r := newRouter("POST", "/auth/verify-otp", VerifyOTP)
	w := doRequest(r, "POST", "/auth/verify-otp", map[string]interface{}{
		"phone": "9876543210",
		"otp":   "000000",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid OTP, got %d", w.Code)
	}
}

func TestVerifyOTP_NewUser(t *testing.T) {
	requireDB(t)
	dropCollection(t, "otps")
	dropCollection(t, "users")

	phone := "9123456701"
	otp := "111222"

	// Pre-insert a valid OTP record
	hash, _ := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.MinCost)
	ctx := context.Background()
	config.GetCollection("otps").InsertOne(ctx, map[string]interface{}{
		"phone":     phone,
		"otpHash":   string(hash),
		"createdAt": time.Now(),
	})

	r := newRouter("POST", "/auth/verify-otp", VerifyOTP)
	w := doRequest(r, "POST", "/auth/verify-otp", map[string]interface{}{
		"phone": phone,
		"otp":   otp,
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	if data["isNewUser"] != true {
		t.Errorf("expected isNewUser=true for unknown phone, got %v", data["isNewUser"])
	}
	if data["tempToken"] == nil {
		t.Error("expected tempToken in response for new user")
	}
}

func TestVerifyOTP_ExistingUser(t *testing.T) {
	requireDB(t)
	dropCollection(t, "otps")
	dropCollection(t, "users")

	phone := "9123456702"
	otp := "333444"

	// Insert existing user
	ctx := context.Background()
	existingUser := models.User{
		ID:        primitive.NewObjectID(),
		Name:      "Existing User",
		Phone:     phone,
		ClassLevel: "10",
		State:     "Delhi",
		Streak:    5,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	config.GetCollection("users").InsertOne(ctx, existingUser)

	// Insert OTP
	hash, _ := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.MinCost)
	config.GetCollection("otps").InsertOne(ctx, map[string]interface{}{
		"phone":     phone,
		"otpHash":   string(hash),
		"createdAt": time.Now(),
	})

	r := newRouter("POST", "/auth/verify-otp", VerifyOTP)
	w := doRequest(r, "POST", "/auth/verify-otp", map[string]interface{}{
		"phone": phone,
		"otp":   otp,
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	if data["isNewUser"] != false {
		t.Errorf("expected isNewUser=false for existing user, got %v", data["isNewUser"])
	}
	if data["token"] == nil {
		t.Error("expected full token for existing user")
	}
}

// ─── Signup ───────────────────────────────────────────────────────────────────

func TestSignup_MissingName(t *testing.T) {
	r := newRouter("POST", "/auth/signup",
		setUserID("", "9876543210"),
		Signup,
	)
	w := doRequest(r, "POST", "/auth/signup", map[string]interface{}{
		"classLevel": "10",
		"state":      "Delhi",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestSignup_MissingClassLevel(t *testing.T) {
	r := newRouter("POST", "/auth/signup",
		setUserID("", "9876543210"),
		Signup,
	)
	w := doRequest(r, "POST", "/auth/signup", map[string]interface{}{
		"name":  "Test User",
		"state": "Delhi",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestSignup_MissingState(t *testing.T) {
	r := newRouter("POST", "/auth/signup",
		setUserID("", "9876543210"),
		Signup,
	)
	w := doRequest(r, "POST", "/auth/signup", map[string]interface{}{
		"name":       "Test User",
		"classLevel": "10",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestSignup_MissingAllFields(t *testing.T) {
	r := newRouter("POST", "/auth/signup",
		setUserID("", "9876543210"),
		Signup,
	)
	w := doRequest(r, "POST", "/auth/signup", map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestSignup_UserAlreadyExists(t *testing.T) {
	requireDB(t)
	dropCollection(t, "users")

	phone := "9123456703"
	ctx := context.Background()
	config.GetCollection("users").InsertOne(ctx, models.User{
		ID:        primitive.NewObjectID(),
		Name:      "Already Here",
		Phone:     phone,
		ClassLevel: "10",
		State:     "UP",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	r := newRouter("POST", "/auth/signup",
		setUserID("", phone),
		Signup,
	)
	w := doRequest(r, "POST", "/auth/signup", map[string]interface{}{
		"name":       "Duplicate",
		"classLevel": "10",
		"state":      "UP",
	}, nil)

	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for existing user, got %d", w.Code)
	}
	body := parseBody(t, w)
	if body["error"] != "USER_EXISTS" {
		t.Errorf("expected USER_EXISTS error code, got %v", body["error"])
	}
}

func TestSignup_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "users")

	phone := "9123456704"
	r := newRouter("POST", "/auth/signup",
		setUserID("", phone),
		Signup,
	)
	w := doRequest(r, "POST", "/auth/signup", map[string]interface{}{
		"name":       "New Student",
		"classLevel": "9",
		"state":      "Maharashtra",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	if data["token"] == nil {
		t.Error("expected token in response")
	}
}

func TestSignup_TokenIsValid(t *testing.T) {
	requireDB(t)
	dropCollection(t, "users")

	phone := "9123456705"
	r := newRouter("POST", "/auth/signup",
		setUserID("", phone),
		Signup,
	)
	w := doRequest(r, "POST", "/auth/signup", map[string]interface{}{
		"name":       "Token Test",
		"classLevel": "8",
		"state":      "Karnataka",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d", w.Code)
	}
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	token, ok := data["token"].(string)
	if !ok || token == "" {
		t.Error("expected non-empty token string")
	}
}

func TestSignup_UserDataInResponse(t *testing.T) {
	requireDB(t)
	dropCollection(t, "users")

	phone := "9123456706"
	r := newRouter("POST", "/auth/signup",
		setUserID("", phone),
		Signup,
	)
	w := doRequest(r, "POST", "/auth/signup", map[string]interface{}{
		"name":       "Ramesh Kumar",
		"classLevel": "12",
		"state":      "Bihar",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Fatalf("want 201, got %d", w.Code)
	}
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	user, ok := data["user"].(map[string]interface{})
	if !ok {
		t.Fatal("expected user object in response")
	}
	if user["name"] != "Ramesh Kumar" {
		t.Errorf("name: want Ramesh Kumar, got %v", user["name"])
	}
	if user["classLevel"] != "12" {
		t.Errorf("classLevel: want 12, got %v", user["classLevel"])
	}
	if user["state"] != "Bihar" {
		t.Errorf("state: want Bihar, got %v", user["state"])
	}
	if user["streak"].(float64) != 1 {
		t.Errorf("streak: want 1, got %v", user["streak"])
	}
}

func TestSignup_PhoneRegex_StartsWith6(t *testing.T) {
	requireDB(t)
	dropCollection(t, "users")

	phone := "6987654321"
	r := newRouter("POST", "/auth/signup",
		setUserID("", phone),
		Signup,
	)
	w := doRequest(r, "POST", "/auth/signup", map[string]interface{}{
		"name": "Test", "classLevel": "10", "state": "Goa",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201 for phone starting with 6, got %d", w.Code)
	}
}
