package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// APIClient handles the communication between the Worner CLI and the Cascata Orchestrator.
type APIClient struct {
	BaseURL string
	Token   string
	Client  *http.Client
}

func NewAPIClient(cfg CommandConfig) *APIClient {
	// In production, BaseURL comes from CASCATA_ADDR or ~/.cascata/config
	addr := os.Getenv("CASCATA_ADDR")
	if addr == "" { addr = "http://localhost:8080" }
	
	token := os.Getenv("CASCATA_TOKEN")
	// If token is missing, we check ~/.cascata/auth.json later in LoadContext
	
	return &APIClient{
		BaseURL: addr,
		Token:   token,
		Client:  &http.Client{Timeout: 30 * time.Second},
	}
}

// Do performs an authenticated request and returns the parsed JSON response.
func (c *APIClient) Do(method, path string, body interface{}, response interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		bodyReader = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, c.BaseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("api.client: request creation failed: %w", err)
	}

	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("api.client: network failure: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		var errResp struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&errResp)
		return fmt.Errorf("api.client: %s: %s (Status: %d)", errResp.Code, errResp.Message, resp.StatusCode)
	}

	if response != nil {
		return json.NewDecoder(resp.Body).Decode(response)
	}
	return nil
}

// Download performs a request and streams the response to a writer (used for .caf export).
func (c *APIClient) Download(path string, out io.Writer) error {
	req, _ := http.NewRequest("GET", c.BaseURL+path, nil)
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.Client.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("api.client: download failed with status %d", resp.StatusCode)
	}

	_, err = io.Copy(out, resp.Body)
	return err
}
