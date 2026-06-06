package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func NotifyNewUser(name, phone string) error {
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		return nil
	}

	payload, _ := json.Marshal(map[string]string{
		"text": fmt.Sprintf("New user signed up!\n*Name:* %s\n*Phone:* %s", name, phone),
	})

	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
