package handlers

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/navodayaprime/api/config"
	"github.com/navodayaprime/api/models"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

// ─── ListAdmins ───────────────────────────────────────────────────────────────

func TestListAdmins_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")

	superID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/manage/admins",
		setAdminID(superID.Hex(), "super@test.com", true),
		ListAdmins,
	)
	w := doRequest(r, "GET", "/admin/manage/admins", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
}

func TestListAdmins_WithAdmins(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")

	ctx := context.Background()
	for i := 0; i < 3; i++ {
		config.GetCollection("admins").InsertOne(ctx, models.Admin{
			ID:        primitive.NewObjectID(),
			FirstName: "Admin",
			LastName:  "User",
			Email:     "admin@test.com",
			IsActive:  true,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		})
	}

	superID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/manage/admins",
		setAdminID(superID.Hex(), "super@test.com", true),
		ListAdmins,
	)
	w := doRequest(r, "GET", "/admin/manage/admins", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	admins := data["admins"].([]interface{})
	if len(admins) != 3 {
		t.Errorf("expected 3 admins, got %d", len(admins))
	}
}

// ─── DeleteAdmin ──────────────────────────────────────────────────────────────

func TestDeleteAdmin_CannotDeleteSelf(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/manage/admins/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteAdmin,
	)
	w := doRequest(r, "DELETE", "/admin/manage/admins/"+superID.Hex(), nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for deleting self, got %d", w.Code)
	}
}

func TestDeleteAdmin_InvalidID(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/manage/admins/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteAdmin,
	)
	w := doRequest(r, "DELETE", "/admin/manage/admins/bad-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid ID, got %d", w.Code)
	}
}

func TestDeleteAdmin_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")

	superID := primitive.NewObjectID()
	targetID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/manage/admins/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteAdmin,
	)
	w := doRequest(r, "DELETE", "/admin/manage/admins/"+targetID.Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404 for non-existent admin, got %d", w.Code)
	}
}

func TestDeleteAdmin_CannotDeleteLastSuperAdmin(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")

	// Insert only one super admin
	superAdmin := models.Admin{
		ID:           primitive.NewObjectID(),
		FirstName:    "Super",
		LastName:     "Admin",
		Email:        "super@test.com",
		IsSuperAdmin: true,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	ctx := context.Background()
	config.GetCollection("admins").InsertOne(ctx, superAdmin)

	// Another super admin tries to delete the only super admin
	callerID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/manage/admins/:id",
		setAdminID(callerID.Hex(), "other@test.com", true),
		DeleteAdmin,
	)
	w := doRequest(r, "DELETE", "/admin/manage/admins/"+superAdmin.ID.Hex(), nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for deleting last super admin, got %d", w.Code)
	}
	body := parseBody(t, w)
	if body["error"] != "LAST_SUPER_ADMIN" {
		t.Errorf("expected LAST_SUPER_ADMIN error, got %v", body["error"])
	}
}

func TestDeleteAdmin_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")

	regularAdmin := models.Admin{
		ID:           primitive.NewObjectID(),
		FirstName:    "Regular",
		Email:        "regular@test.com",
		IsSuperAdmin: false,
		IsActive:     true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	ctx := context.Background()
	config.GetCollection("admins").InsertOne(ctx, regularAdmin)

	superID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/manage/admins/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteAdmin,
	)
	w := doRequest(r, "DELETE", "/admin/manage/admins/"+regularAdmin.ID.Hex(), nil, nil)
	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── InviteAdmin ──────────────────────────────────────────────────────────────

func TestInviteAdmin_MissingFields(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/manage/admins/invite",
		setAdminID(superID.Hex(), "super@test.com", true),
		InviteAdmin,
	)
	w := doRequest(r, "POST", "/admin/manage/admins/invite", map[string]interface{}{
		"firstName": "Test",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing fields, got %d", w.Code)
	}
}

func TestInviteAdmin_InvalidEmail(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/manage/admins/invite",
		setAdminID(superID.Hex(), "super@test.com", true),
		InviteAdmin,
	)
	w := doRequest(r, "POST", "/admin/manage/admins/invite", map[string]interface{}{
		"firstName": "Test",
		"lastName":  "User",
		"email":     "not-an-email",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid email, got %d", w.Code)
	}
}

func TestInviteAdmin_EmailAlreadyExists(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")

	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.MinCost)
	ctx := context.Background()
	config.GetCollection("admins").InsertOne(ctx, models.Admin{
		ID:        primitive.NewObjectID(),
		FirstName: "Existing",
		Email:     "existing@test.com",
		Password:  string(hash),
		IsActive:  true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	})

	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/manage/admins/invite",
		setAdminID(superID.Hex(), "super@test.com", true),
		InviteAdmin,
	)
	w := doRequest(r, "POST", "/admin/manage/admins/invite", map[string]interface{}{
		"firstName": "Another",
		"lastName":  "Admin",
		"email":     "existing@test.com",
	}, nil)
	if w.Code != http.StatusConflict {
		t.Errorf("want 409 for duplicate email, got %d", w.Code)
	}
}

func TestInviteAdmin_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")

	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/manage/admins/invite",
		setAdminID(superID.Hex(), "super@test.com", true),
		InviteAdmin,
	)
	w := doRequest(r, "POST", "/admin/manage/admins/invite", map[string]interface{}{
		"firstName":    "New",
		"lastName":     "Admin",
		"email":        "newadmin@test.com",
		"isSuperAdmin": false,
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	body := parseBody(t, w)
	if body["success"] != true {
		t.Error("expected success=true")
	}
	data := body["data"].(map[string]interface{})
	if data["tempPassword"] == nil || data["tempPassword"] == "" {
		t.Error("expected tempPassword in response")
	}
}

func TestInviteAdmin_SuperAdminInvite(t *testing.T) {
	requireDB(t)
	dropCollection(t, "admins")

	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/manage/admins/invite",
		setAdminID(superID.Hex(), "super@test.com", true),
		InviteAdmin,
	)
	w := doRequest(r, "POST", "/admin/manage/admins/invite", map[string]interface{}{
		"firstName":    "Super",
		"lastName":     "Two",
		"email":        "super2@test.com",
		"isSuperAdmin": true,
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d", w.Code)
	}
}
