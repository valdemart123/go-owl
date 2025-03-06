package browsers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"time"
)

// WebKit struct for Safari automation
type WebKit struct {
	cmd       *exec.Cmd
	sessionID string
}

// Launch starts a new Safari (WebKit) instance using safaridriver
func (w *WebKit) Launch() error {
	log.Println("Launching Safari (WebKit)...")

	// Ensure WebKit automation is enabled
	enableCmd := exec.Command("safaridriver", "--enable")
	if err := enableCmd.Run(); err != nil {
		return fmt.Errorf("failed to enable Safari WebDriver: %w", err)
	}

	// Start safaridriver on a specified port (e.g., 4444)
	w.cmd = exec.Command("safaridriver", "--port=4444")
	if err := w.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Safari WebDriver: %w", err)
	}

	// Wait briefly to ensure safaridriver is ready
	time.Sleep(2 * time.Second)

	// Create a new session
	sessionID, err := w.createSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	w.sessionID = sessionID
	log.Println("Safari session created:", w.sessionID)
	return nil
}

// createSession sends a request to safaridriver to create a new session
func (w *WebKit) createSession() (string, error) {
	url := "http://localhost:4444/session"
	payload := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"alwaysMatch": map[string]interface{}{
				"browserName": "safari",
			},
		},
	}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode session response: %w", err)
	}

	// Expecting the session response in the form: { "value": { "sessionId": "..." } }
	sessionData, ok := result["value"].(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid session response format")
	}
	sessionID, ok := sessionData["sessionId"].(string)
	if !ok {
		return "", fmt.Errorf("invalid session response format: sessionId not found")
	}

	return sessionID, nil
}

// OpenURL navigates to the given URL in the active Safari session
func (w *WebKit) OpenURL(urlStr string) error {
	if w.sessionID == "" {
		return fmt.Errorf("no active session, start Safari first")
	}

	requestURL := fmt.Sprintf("http://localhost:4444/session/%s/url", w.sessionID)
	payload := map[string]string{"url": urlStr}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(requestURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to open URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to navigate to URL, status code: %d", resp.StatusCode)
	}

	log.Println("Opened URL in Safari:", urlStr)
	return nil
}

// Close shuts down the Safari WebDriver instance
func (w *WebKit) Close() error {
	if w.cmd != nil && w.cmd.Process != nil {
		if err := w.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to close Safari WebDriver: %w", err)
		}
		log.Println("Safari WebDriver closed successfully.")
	}
	return nil
}
