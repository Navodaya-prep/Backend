package utils

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/navodayaprime/api/models"

	"go.mongodb.org/mongo-driver/bson"
	"golang.org/x/crypto/bcrypt"
)

// ─── GenerateOTP ──────────────────────────────────────────────────────────────

func TestGenerateOTP_Length(t *testing.T) {
	otp := GenerateOTP()
	if len(otp) != 6 {
		t.Errorf("expected 6-digit OTP, got %d chars: %q", len(otp), otp)
	}
}

func TestGenerateOTP_AllDigits(t *testing.T) {
	for i := 0; i < 100; i++ {
		otp := GenerateOTP()
		if _, err := strconv.Atoi(otp); err != nil {
			t.Errorf("OTP %q is not all digits: %v", otp, err)
		}
	}
}

func TestGenerateOTP_InRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		otp := GenerateOTP()
		n, _ := strconv.Atoi(otp)
		if n < 0 || n > 999999 {
			t.Errorf("OTP %d out of expected range [0, 999999]", n)
		}
	}
}

func TestGenerateOTP_LeadingZeros(t *testing.T) {
	// The function uses %06d, so it pads to 6 digits with leading zeros
	otp := GenerateOTP()
	if len(otp) != 6 {
		t.Errorf("expected length 6 (including leading zeros), got %d: %q", len(otp), otp)
	}
}

func TestGenerateOTP_NotAlwaysSame(t *testing.T) {
	// Generate many OTPs and verify there's some variation
	seen := make(map[string]bool)
	for i := 0; i < 50; i++ {
		seen[GenerateOTP()] = true
	}
	if len(seen) < 5 {
		t.Errorf("expected variety in generated OTPs, got only %d distinct values in 50 attempts", len(seen))
	}
}

// ─── CreateOTP (requires MongoDB) ────────────────────────────────────────────

func TestCreateOTP_DevMode(t *testing.T) {
	requireDB(t)
	clearCollection(t, "otps")

	// OTP_DEV_MODE=true is set in TestMain, so no SMS will be sent
	err := CreateOTP("9876543210")
	if err != nil {
		t.Fatalf("CreateOTP failed: %v", err)
	}

	// Verify OTP was stored in DB
	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("otps")
	var record models.OTP
	err = col.FindOne(ctx, bson.M{"phone": "9876543210"}).Decode(&record)
	if err != nil {
		t.Fatalf("expected OTP record in DB, got error: %v", err)
	}
	if record.OTPHash == "" {
		t.Error("expected non-empty OTP hash")
	}
	if record.Phone != "9876543210" {
		t.Errorf("expected phone=9876543210, got %s", record.Phone)
	}
}

func TestCreateOTP_Upsert(t *testing.T) {
	requireDB(t)
	clearCollection(t, "otps")

	// Call twice — should upsert (only one record)
	CreateOTP("9876543210")
	CreateOTP("9876543210")

	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("otps")
	count, _ := col.CountDocuments(ctx, bson.M{"phone": "9876543210"})
	if count != 1 {
		t.Errorf("expected 1 OTP record (upsert), got %d", count)
	}
}

func TestCreateOTP_HashIsNotPlaintext(t *testing.T) {
	requireDB(t)
	clearCollection(t, "otps")

	CreateOTP("9876543210")

	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("otps")
	var record models.OTP
	col.FindOne(ctx, bson.M{"phone": "9876543210"}).Decode(&record)

	// OTP hash should be a bcrypt hash, not the raw OTP
	// A valid bcrypt hash starts with "$2a$" or "$2b$"
	if len(record.OTPHash) < 10 {
		t.Error("expected a proper bcrypt hash, got very short string")
	}
}

// ─── VerifyOTP (requires MongoDB) ────────────────────────────────────────────

func TestVerifyOTP_Valid(t *testing.T) {
	requireDB(t)
	clearCollection(t, "otps")

	phone := "9123456780"
	otp := "123456"
	hash, _ := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.MinCost)

	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("otps")
	col.InsertOne(ctx, bson.M{
		"phone":     phone,
		"otpHash":   string(hash),
		"createdAt": time.Now(),
	})

	valid, err := VerifyOTP(phone, otp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("expected valid OTP to return true")
	}
}

func TestVerifyOTP_WrongOTP(t *testing.T) {
	requireDB(t)
	clearCollection(t, "otps")

	phone := "9123456781"
	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.MinCost)

	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("otps")
	col.InsertOne(ctx, bson.M{
		"phone":     phone,
		"otpHash":   string(hash),
		"createdAt": time.Now(),
	})

	valid, err := VerifyOTP(phone, "999999")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected wrong OTP to return false")
	}
}

func TestVerifyOTP_NotFound(t *testing.T) {
	requireDB(t)
	clearCollection(t, "otps")

	valid, err := VerifyOTP("9000000000", "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected false for non-existent phone")
	}
}

func TestVerifyOTP_Expired(t *testing.T) {
	requireDB(t)
	clearCollection(t, "otps")

	phone := "9123456782"
	hash, _ := bcrypt.GenerateFromPassword([]byte("654321"), bcrypt.MinCost)

	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("otps")
	col.InsertOne(ctx, bson.M{
		"phone":     phone,
		"otpHash":   string(hash),
		"createdAt": time.Now().Add(-10 * time.Minute), // expired
	})

	valid, err := VerifyOTP(phone, "654321")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected expired OTP to return false")
	}
}

func TestVerifyOTP_DeletedAfterUse(t *testing.T) {
	requireDB(t)
	clearCollection(t, "otps")

	phone := "9123456783"
	otp := "111111"
	hash, _ := bcrypt.GenerateFromPassword([]byte(otp), bcrypt.MinCost)

	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("otps")
	col.InsertOne(ctx, bson.M{
		"phone":     phone,
		"otpHash":   string(hash),
		"createdAt": time.Now(),
	})

	// First verify should succeed
	valid, _ := VerifyOTP(phone, otp)
	if !valid {
		t.Fatal("expected first verify to succeed")
	}

	// Second verify should fail — OTP was deleted after first use
	valid, _ = VerifyOTP(phone, otp)
	if valid {
		t.Error("expected second verify to fail (OTP already consumed)")
	}
}

func TestVerifyOTP_ExpiredRecordDeleted(t *testing.T) {
	requireDB(t)
	clearCollection(t, "otps")

	phone := "9123456784"
	hash, _ := bcrypt.GenerateFromPassword([]byte("222222"), bcrypt.MinCost)

	ctx := context.Background()
	col := testMongoClient.Database("navodaya_utils_test").Collection("otps")
	col.InsertOne(ctx, bson.M{
		"phone":     phone,
		"otpHash":   string(hash),
		"createdAt": time.Now().Add(-10 * time.Minute),
	})

	VerifyOTP(phone, "222222") // should delete expired record

	count, _ := col.CountDocuments(ctx, bson.M{"phone": phone})
	if count != 0 {
		t.Errorf("expected expired OTP record to be deleted, found %d records", count)
	}
}
