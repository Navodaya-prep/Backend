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

func insertTeacher(t *testing.T, email string, isActive bool) models.Teacher {
	t.Helper()
	hash, _ := bcrypt.GenerateFromPassword([]byte("pass123"), bcrypt.MinCost)
	teacher := models.Teacher{
		ID:        primitive.NewObjectID(),
		FirstName: "Test",
		LastName:  "Teacher",
		Email:     email,
		Password:  string(hash),
		IsActive:  isActive,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	config.GetCollection("teachers").InsertOne(context.Background(), teacher)
	return teacher
}

// ─── ListTeachers ─────────────────────────────────────────────────────────────

func TestListTeachers_Empty(t *testing.T) {
	requireDB(t)
	dropCollection(t, "teachers")

	superID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/manage/teachers",
		setAdminID(superID.Hex(), "super@test.com", true),
		ListTeachers,
	)
	w := doRequest(r, "GET", "/admin/manage/teachers", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	teachers, ok := data["teachers"].([]interface{})
	if !ok {
		t.Fatal("expected teachers array")
	}
	if len(teachers) != 0 {
		t.Errorf("expected 0 teachers, got %d", len(teachers))
	}
}

func TestListTeachers_WithData(t *testing.T) {
	requireDB(t)
	dropCollection(t, "teachers")

	insertTeacher(t, "t1@test.com", true)
	insertTeacher(t, "t2@test.com", true)

	superID := primitive.NewObjectID()
	r := newRouter("GET", "/admin/manage/teachers",
		setAdminID(superID.Hex(), "super@test.com", true),
		ListTeachers,
	)
	w := doRequest(r, "GET", "/admin/manage/teachers", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d", w.Code)
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	teachers := data["teachers"].([]interface{})
	if len(teachers) != 2 {
		t.Errorf("expected 2 teachers, got %d", len(teachers))
	}
}

// ─── InviteTeacher ────────────────────────────────────────────────────────────

func TestInviteTeacher_MissingFields(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/manage/teachers/invite",
		setAdminID(superID.Hex(), "super@test.com", true),
		InviteTeacher,
	)
	w := doRequest(r, "POST", "/admin/manage/teachers/invite", map[string]interface{}{
		"firstName": "Only",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing fields, got %d", w.Code)
	}
}

func TestInviteTeacher_InvalidEmail(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/manage/teachers/invite",
		setAdminID(superID.Hex(), "super@test.com", true),
		InviteTeacher,
	)
	w := doRequest(r, "POST", "/admin/manage/teachers/invite", map[string]interface{}{
		"firstName": "Test",
		"lastName":  "Teacher",
		"email":     "invalid",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for invalid email, got %d", w.Code)
	}
}

func TestInviteTeacher_DuplicateEmail(t *testing.T) {
	requireDB(t)
	dropCollection(t, "teachers")

	insertTeacher(t, "dup@test.com", true)

	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/manage/teachers/invite",
		setAdminID(superID.Hex(), "super@test.com", true),
		InviteTeacher,
	)
	w := doRequest(r, "POST", "/admin/manage/teachers/invite", map[string]interface{}{
		"firstName": "Test",
		"lastName":  "Teacher",
		"email":     "dup@test.com",
	}, nil)
	if w.Code != http.StatusConflict {
		t.Errorf("want 409 for duplicate email, got %d", w.Code)
	}
}

func TestInviteTeacher_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "teachers")

	superID := primitive.NewObjectID()
	r := newRouter("POST", "/admin/manage/teachers/invite",
		setAdminID(superID.Hex(), "super@test.com", true),
		InviteTeacher,
	)
	w := doRequest(r, "POST", "/admin/manage/teachers/invite", map[string]interface{}{
		"firstName": "New",
		"lastName":  "Teacher",
		"email":     "newteacher@test.com",
		"subject":   "Maths",
	}, nil)

	if w.Code != http.StatusCreated {
		t.Errorf("want 201, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	if data["tempPassword"] == nil {
		t.Error("expected tempPassword in response")
	}
}

// ─── UpdateTeacher ────────────────────────────────────────────────────────────

func TestUpdateTeacher_InvalidID(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/manage/teachers/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		UpdateTeacher,
	)
	w := doRequest(r, "PUT", "/admin/manage/teachers/bad-id", map[string]interface{}{
		"firstName": "A",
		"lastName":  "B",
	}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestUpdateTeacher_MissingFields(t *testing.T) {
	superID := primitive.NewObjectID()
	teacherID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/manage/teachers/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		UpdateTeacher,
	)
	w := doRequest(r, "PUT", "/admin/manage/teachers/"+teacherID.Hex(), map[string]interface{}{}, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400 for missing firstName/lastName, got %d", w.Code)
	}
}

func TestUpdateTeacher_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "teachers")

	superID := primitive.NewObjectID()
	teacherID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/manage/teachers/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		UpdateTeacher,
	)
	w := doRequest(r, "PUT", "/admin/manage/teachers/"+teacherID.Hex(), map[string]interface{}{
		"firstName": "New",
		"lastName":  "Name",
	}, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestUpdateTeacher_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "teachers")

	teacher := insertTeacher(t, "update@test.com", true)

	superID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/manage/teachers/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		UpdateTeacher,
	)
	w := doRequest(r, "PUT", "/admin/manage/teachers/"+teacher.ID.Hex(), map[string]interface{}{
		"firstName": "Updated",
		"lastName":  "Teacher",
		"subject":   "Science",
	}, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}

// ─── ToggleTeacherStatus ──────────────────────────────────────────────────────

func TestToggleTeacherStatus_InvalidID(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/manage/teachers/:id/toggle",
		setAdminID(superID.Hex(), "super@test.com", true),
		ToggleTeacherStatus,
	)
	w := doRequest(r, "PUT", "/admin/manage/teachers/bad-id/toggle", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestToggleTeacherStatus_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "teachers")

	superID := primitive.NewObjectID()
	teacherID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/manage/teachers/:id/toggle",
		setAdminID(superID.Hex(), "super@test.com", true),
		ToggleTeacherStatus,
	)
	w := doRequest(r, "PUT", "/admin/manage/teachers/"+teacherID.Hex()+"/toggle", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestToggleTeacherStatus_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "teachers")

	teacher := insertTeacher(t, "toggle@test.com", true) // active = true

	superID := primitive.NewObjectID()
	r := newRouter("PUT", "/admin/manage/teachers/:id/toggle",
		setAdminID(superID.Hex(), "super@test.com", true),
		ToggleTeacherStatus,
	)
	w := doRequest(r, "PUT", "/admin/manage/teachers/"+teacher.ID.Hex()+"/toggle", nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
	data := parseBody(t, w)["data"].(map[string]interface{})
	if data["isActive"] != false {
		t.Errorf("expected isActive=false after toggling active teacher, got %v", data["isActive"])
	}
}

// ─── DeleteTeacher ────────────────────────────────────────────────────────────

func TestDeleteTeacher_InvalidID(t *testing.T) {
	superID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/manage/teachers/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteTeacher,
	)
	w := doRequest(r, "DELETE", "/admin/manage/teachers/bad-id", nil, nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("want 400, got %d", w.Code)
	}
}

func TestDeleteTeacher_NotFound(t *testing.T) {
	requireDB(t)
	dropCollection(t, "teachers")

	superID := primitive.NewObjectID()
	teacherID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/manage/teachers/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteTeacher,
	)
	w := doRequest(r, "DELETE", "/admin/manage/teachers/"+teacherID.Hex(), nil, nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("want 404, got %d", w.Code)
	}
}

func TestDeleteTeacher_Success(t *testing.T) {
	requireDB(t)
	dropCollection(t, "teachers")

	teacher := insertTeacher(t, "delete@test.com", true)

	superID := primitive.NewObjectID()
	r := newRouter("DELETE", "/admin/manage/teachers/:id",
		setAdminID(superID.Hex(), "super@test.com", true),
		DeleteTeacher,
	)
	w := doRequest(r, "DELETE", "/admin/manage/teachers/"+teacher.ID.Hex(), nil, nil)

	if w.Code != http.StatusOK {
		t.Errorf("want 200, got %d: %s", w.Code, w.Body.String())
	}
}
