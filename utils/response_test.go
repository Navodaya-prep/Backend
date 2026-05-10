package utils

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func newTestContext() (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)
	return c, w
}

// ─── Success ─────────────────────────────────────────────────────────────────

func TestSuccess_WithData(t *testing.T) {
	c, w := newTestContext()
	data := gin.H{"key": "value", "num": 42}
	Success(c, http.StatusOK, data, "operation done")

	if w.Code != http.StatusOK {
		t.Errorf("status: want 200, got %d", w.Code)
	}

	var resp APIResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !resp.Success {
		t.Error("expected Success=true")
	}
	if resp.Message != "operation done" {
		t.Errorf("Message: want 'operation done', got %q", resp.Message)
	}
	if resp.Data == nil {
		t.Error("expected non-nil Data")
	}
}

func TestSuccess_WithNilData(t *testing.T) {
	c, w := newTestContext()
	Success(c, http.StatusOK, nil, "no data")

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if !resp.Success {
		t.Error("expected Success=true")
	}
	if resp.Data != nil {
		t.Errorf("expected nil Data for omitempty, got %v", resp.Data)
	}
}

func TestSuccess_StatusCreated(t *testing.T) {
	c, w := newTestContext()
	Success(c, http.StatusCreated, gin.H{"id": "abc"}, "created")

	if w.Code != http.StatusCreated {
		t.Errorf("status: want 201, got %d", w.Code)
	}
}

func TestSuccess_StatusNoContent(t *testing.T) {
	c, w := newTestContext()
	Success(c, http.StatusNoContent, nil, "")

	if w.Code != http.StatusNoContent {
		t.Errorf("status: want 204, got %d", w.Code)
	}
}

func TestSuccess_EmptyMessage(t *testing.T) {
	c, w := newTestContext()
	Success(c, http.StatusOK, nil, "")

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Message != "" {
		t.Errorf("expected empty message, got %q", resp.Message)
	}
}

func TestSuccess_SliceData(t *testing.T) {
	c, w := newTestContext()
	data := []string{"a", "b", "c"}
	Success(c, http.StatusOK, data, "list")

	var resp APIResponse
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Data == nil {
		t.Error("expected non-nil Data for slice")
	}
}

func TestSuccess_ContentTypeJSON(t *testing.T) {
	c, w := newTestContext()
	Success(c, http.StatusOK, nil, "ok")

	ct := w.Header().Get("Content-Type")
	if ct == "" {
		t.Error("expected Content-Type header to be set")
	}
}

// ─── ErrorRes ────────────────────────────────────────────────────────────────

func TestErrorRes_BadRequest(t *testing.T) {
	c, w := newTestContext()
	ErrorRes(c, http.StatusBadRequest, "INVALID_INPUT", "field missing")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status: want 400, got %d", w.Code)
	}

	var resp APIError
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if resp.Success {
		t.Error("expected Success=false")
	}
	if resp.Error != "INVALID_INPUT" {
		t.Errorf("Error: want INVALID_INPUT, got %q", resp.Error)
	}
	if resp.Message != "field missing" {
		t.Errorf("Message: want 'field missing', got %q", resp.Message)
	}
}

func TestErrorRes_Unauthorized(t *testing.T) {
	c, w := newTestContext()
	ErrorRes(c, http.StatusUnauthorized, "UNAUTHORIZED", "auth required")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status: want 401, got %d", w.Code)
	}

	var resp APIError
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Error != "UNAUTHORIZED" {
		t.Errorf("Error: want UNAUTHORIZED, got %q", resp.Error)
	}
}

func TestErrorRes_Forbidden(t *testing.T) {
	c, w := newTestContext()
	ErrorRes(c, http.StatusForbidden, "FORBIDDEN", "access denied")

	if w.Code != http.StatusForbidden {
		t.Errorf("status: want 403, got %d", w.Code)
	}
}

func TestErrorRes_NotFound(t *testing.T) {
	c, w := newTestContext()
	ErrorRes(c, http.StatusNotFound, "NOT_FOUND", "resource missing")

	if w.Code != http.StatusNotFound {
		t.Errorf("status: want 404, got %d", w.Code)
	}
}

func TestErrorRes_InternalServer(t *testing.T) {
	c, w := newTestContext()
	ErrorRes(c, http.StatusInternalServerError, "DB_ERROR", "database failed")

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status: want 500, got %d", w.Code)
	}

	var resp APIError
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Success {
		t.Error("expected Success=false")
	}
}

func TestErrorRes_EmptyFields(t *testing.T) {
	c, w := newTestContext()
	ErrorRes(c, http.StatusBadRequest, "", "")

	var resp APIError
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Success {
		t.Error("expected Success=false")
	}
}

// ─── APIResponse struct ───────────────────────────────────────────────────────

func TestAPIResponse_SuccessField(t *testing.T) {
	r := APIResponse{Success: true, Message: "ok", Data: 42}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var out map[string]interface{}
	json.Unmarshal(b, &out)
	if out["success"] != true {
		t.Error("expected success=true in JSON")
	}
	if out["message"] != "ok" {
		t.Errorf("expected message=ok, got %v", out["message"])
	}
}

func TestAPIResponse_OmitsNilData(t *testing.T) {
	r := APIResponse{Success: true, Message: "ok", Data: nil}
	b, _ := json.Marshal(r)
	var out map[string]interface{}
	json.Unmarshal(b, &out)
	if _, exists := out["data"]; exists {
		t.Error("expected data field to be omitted when nil (omitempty)")
	}
}

// ─── APIError struct ──────────────────────────────────────────────────────────

func TestAPIError_Fields(t *testing.T) {
	e := APIError{Success: false, Error: "ERR_CODE", Message: "something went wrong"}
	b, _ := json.Marshal(e)
	var out map[string]interface{}
	json.Unmarshal(b, &out)
	if out["success"] != false {
		t.Error("expected success=false")
	}
	if out["error"] != "ERR_CODE" {
		t.Errorf("expected error=ERR_CODE, got %v", out["error"])
	}
	if out["message"] != "something went wrong" {
		t.Errorf("expected proper message, got %v", out["message"])
	}
}
