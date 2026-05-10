package utils

import (
	"os"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ─── SignToken ────────────────────────────────────────────────────────────────

func TestSignToken_Success(t *testing.T) {
	token, err := SignToken("user123", "9876543210")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
}

func TestSignToken_ClaimsRoundTrip(t *testing.T) {
	token, err := SignToken("user123", "9876543210")
	if err != nil {
		t.Fatalf("SignToken error: %v", err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if claims.UserID != "user123" {
		t.Errorf("UserID: want user123, got %s", claims.UserID)
	}
	if claims.Phone != "9876543210" {
		t.Errorf("Phone: want 9876543210, got %s", claims.Phone)
	}
	if claims.IsTemp {
		t.Error("expected IsTemp=false for a full token")
	}
}

func TestSignToken_EmptyUserID(t *testing.T) {
	token, err := SignToken("", "9876543210")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if claims.UserID != "" {
		t.Errorf("expected empty UserID, got %s", claims.UserID)
	}
}

func TestSignToken_ExpiryFuture(t *testing.T) {
	token, _ := SignToken("user123", "9876543210")
	claims, _ := ParseToken(token)
	if claims.ExpiresAt == nil {
		t.Fatal("expected non-nil ExpiresAt")
	}
	if !claims.ExpiresAt.Time.After(time.Now()) {
		t.Error("expected token expiry to be in the future")
	}
}

// ─── SignTempToken ────────────────────────────────────────────────────────────

func TestSignTempToken_IsTemp(t *testing.T) {
	token, err := SignTempToken("9876543210")
	if err != nil {
		t.Fatalf("SignTempToken error: %v", err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if !claims.IsTemp {
		t.Error("expected IsTemp=true")
	}
	if claims.Phone != "9876543210" {
		t.Errorf("Phone: want 9876543210, got %s", claims.Phone)
	}
}

func TestSignTempToken_ShortExpiry(t *testing.T) {
	token, _ := SignTempToken("9876543210")
	claims, _ := ParseToken(token)
	if claims.ExpiresAt == nil {
		t.Fatal("expected non-nil ExpiresAt")
	}
	// Should expire in ~10 minutes, so definitely within 11 minutes from now
	maxExpiry := time.Now().Add(11 * time.Minute)
	if claims.ExpiresAt.Time.After(maxExpiry) {
		t.Errorf("temp token expiry too far in future: %v", claims.ExpiresAt.Time)
	}
}

func TestSignTempToken_EmptyUserID(t *testing.T) {
	token, err := SignTempToken("9876543210")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	claims, err := ParseToken(token)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if claims.UserID != "" {
		t.Errorf("expected empty UserID in temp token, got %s", claims.UserID)
	}
}

// ─── ParseToken ──────────────────────────────────────────────────────────────

func TestParseToken_Valid(t *testing.T) {
	raw, _ := SignToken("u1", "1111111111")
	claims, err := ParseToken(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != "u1" {
		t.Errorf("UserID: want u1, got %s", claims.UserID)
	}
}

func TestParseToken_EmptyString(t *testing.T) {
	_, err := ParseToken("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestParseToken_GarbageString(t *testing.T) {
	_, err := ParseToken("not.a.valid.jwt")
	if err == nil {
		t.Fatal("expected error for garbage token")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	claims := Claims{
		UserID: "u1",
		Phone:  "9876543210",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := tok.SignedString([]byte("completely-different-secret"))

	_, err := ParseToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for wrong signing key")
	}
}

func TestParseToken_Expired(t *testing.T) {
	claims := Claims{
		UserID: "u1",
		Phone:  "9876543210",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := tok.SignedString([]byte(os.Getenv("JWT_SECRET")))

	_, err := ParseToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestParseToken_WrongSigningMethod(t *testing.T) {
	claims := Claims{UserID: "u1"}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, _ := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := ParseToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for unsigned (none) token")
	}
}

// ─── SignAdminToken ───────────────────────────────────────────────────────────

func TestSignAdminToken_RegularAdmin(t *testing.T) {
	token, err := SignAdminToken("admin123", "admin@test.com", false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	claims, err := ParseAdminToken(token)
	if err != nil {
		t.Fatalf("ParseAdminToken error: %v", err)
	}
	if claims.AdminID != "admin123" {
		t.Errorf("AdminID: want admin123, got %s", claims.AdminID)
	}
	if claims.Email != "admin@test.com" {
		t.Errorf("Email: want admin@test.com, got %s", claims.Email)
	}
	if claims.IsSuperAdmin {
		t.Error("expected IsSuperAdmin=false")
	}
}

func TestSignAdminToken_SuperAdmin(t *testing.T) {
	token, err := SignAdminToken("super1", "super@test.com", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	claims, err := ParseAdminToken(token)
	if err != nil {
		t.Fatalf("ParseAdminToken error: %v", err)
	}
	if !claims.IsSuperAdmin {
		t.Error("expected IsSuperAdmin=true")
	}
}

func TestSignAdminToken_ExpiryFuture(t *testing.T) {
	token, _ := SignAdminToken("a1", "a@test.com", false)
	claims, _ := ParseAdminToken(token)
	if claims.ExpiresAt == nil {
		t.Fatal("expected non-nil ExpiresAt")
	}
	if !claims.ExpiresAt.Time.After(time.Now()) {
		t.Error("expected admin token expiry to be in the future")
	}
}

// ─── ParseAdminToken ──────────────────────────────────────────────────────────

func TestParseAdminToken_Valid(t *testing.T) {
	raw, _ := SignAdminToken("a2", "a2@test.com", true)
	claims, err := ParseAdminToken(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.AdminID != "a2" {
		t.Errorf("AdminID: want a2, got %s", claims.AdminID)
	}
	if !claims.IsSuperAdmin {
		t.Error("expected IsSuperAdmin=true")
	}
}

func TestParseAdminToken_EmptyString(t *testing.T) {
	_, err := ParseAdminToken("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestParseAdminToken_GarbageString(t *testing.T) {
	_, err := ParseAdminToken("garbage.token.here")
	if err == nil {
		t.Fatal("expected error for garbage token")
	}
}

func TestParseAdminToken_Expired(t *testing.T) {
	claims := AdminClaims{
		AdminID: "a3",
		Email:   "a3@test.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := tok.SignedString([]byte(os.Getenv("JWT_SECRET")))

	_, err := ParseAdminToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for expired admin token")
	}
}

func TestParseAdminToken_WrongSecret(t *testing.T) {
	claims := AdminClaims{
		AdminID: "a4",
		Email:   "a4@test.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := tok.SignedString([]byte("wrong-secret"))

	_, err := ParseAdminToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for wrong signing key")
	}
}

func TestParseAdminToken_WrongSigningMethod(t *testing.T) {
	claims := AdminClaims{AdminID: "a5"}
	tok := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, _ := tok.SignedString(jwt.UnsafeAllowNoneSignatureType)

	_, err := ParseAdminToken(tokenStr)
	if err == nil {
		t.Fatal("expected error for unsigned (none) admin token")
	}
}
