package utils

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNotifyNewUser_SendsCorrectPayload(t *testing.T) {
	realURL := os.Getenv("SLACK_WEBHOOK_URL")

	if realURL != "" {
		err := NotifyNewUser("testing", "9876543210")
		if err != nil {
			t.Fatalf("Slack error: %v", err)
		}
		t.Log("Message sent to real Slack channel successfully")
		return
	}

	// No real URL set — use a local mock server
	var received map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	os.Setenv("SLACK_WEBHOOK_URL", server.URL)
	defer os.Unsetenv("SLACK_WEBHOOK_URL")

	err := NotifyNewUser("testing", "9876543210")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if received["text"] == "" {
		t.Fatal("expected a text payload, got empty")
	}
	t.Logf("Slack message sent (mock): %s", received["text"])
}

func TestNotifyNewUser_NoWebhookURL(t *testing.T) {
	os.Unsetenv("SLACK_WEBHOOK_URL")

	// Should silently no-op, not error
	err := NotifyNewUser("testing", "9876543210")
	if err != nil {
		t.Fatalf("expected nil error when webhook URL is not set, got: %v", err)
	}
}
