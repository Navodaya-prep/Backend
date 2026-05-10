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

// ─── helpers ─────────────────────────────────────────────────────────────────

func insertAdmin(t *testing.T, email, password string, isSuper, isActive bool) models.Admin {
	t.Helper()
	requireDB(t)

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt error: %v", err)
	}
	admin := models.Admin{
		ID:           primitive.NewObjectID(),
		FirstName:    "Test",
		LastName:     "Admin",
		Email:        email,
		Password:     string(hash),
		IsSuperAdmin: isSuper,
		IsActive:     isActive,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	ctx := context.Background()
	_, err = config.GetCollection("admins").InsertOne(ctx, admin)
	if err != nil {
		t.Fatalf("insert admin: %v", err)
	}
	return admin
}

// ─── AdminLogin ───────────────────────────────────────────────────────────────

func TestAdminLogin_MissingEmail(t *testing.T) {
	r := newRouter("POST", "/admin/auth/login", AdminLogin)
	w := doRequest(r, "POST", "/admin/auth/login", map[string]interface{}{
		"password": "secret123",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestAdminLogin_MissingPassword(t *testing.T) {
	r := newRouter("POST", "/admin/auth/login", AdminLogin)
	w := doRequest(r, "POST", "/admin/auth/login", map[string]interface{}{
		"email": "admin@test.com",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestAdminLogin_MissingBothFields(t *testing.T) {
	r := newRouter("POST", "/admin/auth/login", AdminLogin)
	w := doRequest(r, "POST", "/admin/auth/login", map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestAdminLogin_InvalidEmailFormat(t *testing.T) {
	r := newRouter("POST", "/admin/auth/login", AdminLogin)
	w := doRequest(r, "POST", "/admin/auth/login", map[string]interface{}{
		"email":    "not-an-email",
		"password": "secret123",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid email, got %d", w.Code)
	}
}

func TestAdminLogin_EmailNotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")

	r := newRouter("POST", "/admin/auth/login", AdminLogin)
	w := doRequest(r, "POST", "/admin/auth/login", map[string]interface{}{
		"email":    "nonexistent@test.com",
		"password": "any_password",
	}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401 for unknown email, got %d", w.Code)
	}
}

func TestAdminLogin_InactiveAccount(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	insertAdmin(t, "inactive@test.com", "pass123", false, false)

	r := newRouter("POST", "/admin/auth/login", AdminLogin)
	w := doRequest(r, "POST", "/admin/auth/login", map[string]interface{}{
		"email":    "inactive@test.com",
		"password": "pass123",
	}, nil)
	if w.Code != http.StatusForbidden {
		t.Errorf("want 403 for inactive account, got %d", w.Code)
	}
	body := parseBody(t, w)
	if body["error"] != "ACCOUNT_INACTIVE" {
		t.Errorf("expected ACCOUNT_INACTIVE, got %v", body["error"])
	}
}

func TestAdminLogin_WrongPassword(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	insertAdmin(t, "active@test.com", "correctpassword", false, true)

	r := newRouter("POST", "/admin/auth/login", AdminLogin)
	w := doRequest(r, "POST", "/admin/auth/login", map[string]interface{}{
		"email":    "active@test.com",
		"password": "wrongpassword",
	}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401 for wrong password, got %d", w.Code)
	}
}

func TestAdminLogin_Success_RegularAdmin(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	insertAdmin(t, "regular@test.com", "mypassword", false, true)

	r := newRouter("POST", "/admin/auth/login", AdminLogin)
	w := doRequest(r, "POST", "/admin/auth/login", map[string]interface{}{
		"email":    "regular@test.com",
		"password": "mypassword",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
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

func TestAdminLogin_Success_SuperAdmin(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	insertAdmin(t, "super@test.com", "superpass", true, true)

	r := newRouter("POST", "/admin/auth/login", AdminLogin)
	w := doRequest(r, "POST", "/admin/auth/login", map[string]interface{}{
		"email":    "super@test.com",
		"password": "superpass",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	if data["token"] == nil {
		t.Error("expected token in response")
	}
}

func TestAdminLogin_TokenNotEmpty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	insertAdmin(t, "tokencheck@test.com", "tokenpass", false, true)

	r := newRouter("POST", "/admin/auth/login", AdminLogin)
	w := doRequest(r, "POST", "/admin/auth/login", map[string]interface{}{
		"email":    "tokencheck@test.com",
		"password": "tokenpass",
	}, nil)

	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	token, ok := data["token"].(string)
	if !ok || token == "" {
		t.Error("expected non-empty string token")
	}
}

// ─── GetAdminProfile ──────────────────────────────────────────────────────────

func TestGetAdminProfile_InvalidID(t *testing.T) {
	r := newRouter("GET", "/admin/auth/profile",
		setAdminID("not-a-valid-hex", "admin@test.com", false),
		GetAdminProfile,
	)
	w := doRequest(r, "GET", "/admin/auth/profile", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid admin ID, got %d", w.Code)
	}
}

func TestGetAdminProfile_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")

	r := newRouter("GET", "/admin/auth/profile",
		setAdminID(primitive.NewObjectID().Hex(), "admin@test.com", false),
		GetAdminProfile,
	)
	w := doRequest(r, "GET", "/admin/auth/profile", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestGetAdminProfile_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	admin := insertAdmin(t, "profile@test.com", "pass", false, true)

	r := newRouter("GET", "/admin/auth/profile",
		setAdminID(admin.ID.Hex(), admin.Email, false),
		GetAdminProfile,
	)
	w := doRequest(r, "GET", "/admin/auth/profile", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	data := body["data"].(map[string]interface{})
	if data["admin"] == nil {
		t.Error("expected admin object in response")
	}
}

// ─── UpdateAdminProfile ───────────────────────────────────────────────────────

func TestUpdateAdminProfile_InvalidEmail(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/auth/profile",
		setAdminID(adminID.Hex(), "admin@test.com", false),
		UpdateAdminProfile,
	)
	w := doRequest(r, "PUT", "/admin/auth/profile", map[string]interface{}{
		"email": "not-an-email",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid email, got %d", w.Code)
	}
}

func TestUpdateAdminProfile_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	admin := insertAdmin(t, "update@test.com", "pass", false, true)

	r := newRouter("PUT", "/admin/auth/profile",
		setAdminID(admin.ID.Hex(), admin.Email, false),
		UpdateAdminProfile,
	)
	w := doRequest(r, "PUT", "/admin/auth/profile", map[string]interface{}{
		"firstName": "Updated",
		"lastName":  "Name",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateAdminProfile_EmailConflict(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	admin1 := insertAdmin(t, "admin1@test.com", "pass1", false, true)
	admin2 := insertAdmin(t, "admin2@test.com", "pass2", false, true)

	r := newRouter("PUT", "/admin/auth/profile",
		setAdminID(admin2.ID.Hex(), admin2.Email, false),
		UpdateAdminProfile,
	)
	w := doRequest(r, "PUT", "/admin/auth/profile", map[string]interface{}{
		"email": admin1.Email, // already taken
	}, nil)

	if w.Code != http.StatusConflict {
		t.Errorf("want 409 for duplicate email, got %d", w.Code)
	}
}

func TestUpdateAdminProfile_EmptyBody(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	admin := insertAdmin(t, "emptyupdate@test.com", "pass", false, true)

	r := newRouter("PUT", "/admin/auth/profile",
		setAdminID(admin.ID.Hex(), admin.Email, false),
		UpdateAdminProfile,
	)
	w := doRequest(r, "PUT", "/admin/auth/profile", map[string]interface{}{}, nil)

	// Empty update is a no-op but should succeed
	if w.Code != http.StatusOK {
		t.Errorf("want 200 for empty update, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── ChangeAdminPassword ──────────────────────────────────────────────────────

func TestChangeAdminPassword_MissingFields(t *testing.T) {
	r := newRouter("PUT", "/admin/auth/change-password",
		setAdminID(primitive.NewObjectID().Hex(), "a@test.com", false),
		ChangeAdminPassword,
	)
	w := doRequest(r, "PUT", "/admin/auth/change-password", map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestChangeAdminPassword_WeakNewPassword(t *testing.T) {
	adminID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/auth/change-password",
		setAdminID(adminID.Hex(), "a@test.com", false),
		ChangeAdminPassword,
	)
	w := doRequest(r, "PUT", "/admin/auth/change-password", map[string]interface{}{
		"currentPassword": "oldpass",
		"newPassword":     "abc", // too short
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for short password, got %d", w.Code)
	}
	body := parseBody(t, w)
	if body["error"] != "WEAK_PASSWORD" {
		t.Errorf("expected WEAK_PASSWORD, got %v", body["error"])
	}
}

func TestChangeAdminPassword_AdminNotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")

	r := newRouter("PUT", "/admin/auth/change-password",
		setAdminID(primitive.NewObjectID().Hex(), "notfound@test.com", false),
		ChangeAdminPassword,
	)
	w := doRequest(r, "PUT", "/admin/auth/change-password", map[string]interface{}{
		"currentPassword": "currentpass",
		"newPassword":     "newpassword",
	}, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestChangeAdminPassword_WrongCurrentPassword(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	admin := insertAdmin(t, "pwchange@test.com", "correctpass", false, true)

	r := newRouter("PUT", "/admin/auth/change-password",
		setAdminID(admin.ID.Hex(), admin.Email, false),
		ChangeAdminPassword,
	)
	w := doRequest(r, "PUT", "/admin/auth/change-password", map[string]interface{}{
		"currentPassword": "wrongcurrentpass",
		"newPassword":     "newpassword123",
	}, nil)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("want 401 for wrong current password, got %d", w.Code)
	}
}

func TestChangeAdminPassword_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	admin := insertAdmin(t, "pwsuccess@test.com", "oldpass123", false, true)

	r := newRouter("PUT", "/admin/auth/change-password",
		setAdminID(admin.ID.Hex(), admin.Email, false),
		ChangeAdminPassword,
	)
	w := doRequest(r, "PUT", "/admin/auth/change-password", map[string]interface{}{
		"currentPassword": "oldpass123",
		"newPassword":     "newpass456",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestChangeAdminPassword_ExactlyMinLength(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")
	admin := insertAdmin(t, "minpw@test.com", "exactly6", false, true)

	r := newRouter("PUT", "/admin/auth/change-password",
		setAdminID(admin.ID.Hex(), admin.Email, false),
		ChangeAdminPassword,
	)
	w := doRequest(r, "PUT", "/admin/auth/change-password", map[string]interface{}{
		"currentPassword": "exactly6",
		"newPassword":     "newmin6",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200 for 6-char password, got %d", w.Code)
	}
}
