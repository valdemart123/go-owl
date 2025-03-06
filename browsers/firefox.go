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

// Firefox struct using native WebDriver commands
type Firefox struct {
	cmd      *exec.Cmd
	sessionID string
}

// Launch starts a new Firefox browser instance using Geckodriver
func (f *Firefox) Launch() error {
	log.Println("Launching Firefox...")
	f.cmd = exec.Command("geckodriver", "--port=4444")
	if err := f.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Geckodriver: %w", err)
	}

	// Wait for Geckodriver to start
	time.Sleep(2 * time.Second)

	// Create a new session
	sessionID, err := f.createSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	f.sessionID = sessionID
	log.Println("Firefox session created:", f.sessionID)
	return nil
}

// createSession sends a request to Geckodriver to create a new session
func (f *Firefox) createSession() (string, error) {
	url := "http://localhost:4444/session"
	payload := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"alwaysMatch": map[string]interface{}{
				"browserName": "firefox",
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

	sessionID, ok := result["value"].(map[string]interface{})["sessionId"].(string)
	if !ok {
		return "", fmt.Errorf("invalid session response format")
	}

	return sessionID, nil
}

// OpenURL navigates to a given URL in the currently running Firefox session
func (f *Firefox) OpenURL(url string) error {
	if f.sessionID == "" {
		return fmt.Errorf("no active session, start Firefox first")
	}

	requestURL := fmt.Sprintf("http://localhost:4444/session/%s/url", f.sessionID)
	payload := map[string]string{"url": url}
	jsonPayload, _ := json.Marshal(payload)

	resp, err := http.Post(requestURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return fmt.Errorf("failed to open URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to navigate to URL, status code: %d", resp.StatusCode)
	}

	log.Println("Opened URL in Firefox:", url)
	return nil
}

// Close shuts down the Firefox browser instance
func (f *Firefox) Close() error {
	if f.cmd != nil && f.cmd.Process != nil {
		if err := f.cmd.Process.Kill(); err != nil {
			return fmt.Errorf("failed to close Firefox: %w", err)
		}
		log.Println("Firefox browser closed successfully.")
	}
	return nil
}
