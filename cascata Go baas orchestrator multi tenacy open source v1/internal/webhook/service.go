package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
	"io"

	"cascata/internal/database"
	"cascata/internal/utils"
)

// InboundReceiver represents a configured webhook endpoint.
type InboundReceiver struct {
	ID          string `json:"id"`
	ProjectSlug string `json:"project_slug"`
	Name        string `json:"name"`
	PathSlug    string `json:"path_slug"`
	AuthMethod  string `json:"auth_method"`
	SecretKey   string `json:"secret_key"`
	TargetType  string `json:"target_type"`
	TargetID    string `json:"target_id"`
}

// Service manages the webhook lifecycle and execution logging.
type Service struct {
	repo *database.Repository
}

func NewService(repo *database.Repository) *Service {
	return &Service{repo: repo}
}

// LogRun saves the execution metadata to system.automation_runs.
func (s *Service) LogRun(ctx context.Context, projectSlug, triggerType, triggerID, status string, duration int, input, output interface{}, errMsg string) {
	// Fire-and-forget: we use a background context and swallow errors.
	go func() {
		sql := `
			INSERT INTO system.automation_runs 
			(project_slug, trigger_type, trigger_id, status, execution_time_ms, input_payload, output_result, error_message)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		`
		_, err := s.repo.Pool.Exec(context.Background(), sql, projectSlug, triggerType, triggerID, status, duration, input, output, errMsg)
		if err != nil {
			slog.Warn("webhook: failed to log run", "error", err)
		}
	}()
}

// HandleInbound processes an incoming HTTP webhook request.
func (s *Service) HandleInbound(ctx context.Context, projectSlug, pathSlug string, r *http.Request) (int, string) {
	start := time.Now()
	
	// 1. Resolve receiver configuration
	sql := `
		SELECT id, auth_method, secret_key, target_type, target_id
		FROM system.webhook_receivers 
		WHERE project_slug = $1 AND path_slug = $2 AND is_active = true
	`
	var recv InboundReceiver
	err := s.repo.Pool.QueryRow(ctx, sql, projectSlug, pathSlug).Scan(&recv.ID, &recv.AuthMethod, &recv.SecretKey, &recv.TargetType, &recv.TargetID)
	if err != nil {
		return http.StatusNotFound, "Webhook not found or inactive"
	}

	// 2. Read Payload
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return http.StatusBadRequest, "Malformed request body"
	}

	// 3. HMAC Validation (Security Gateway)
	if recv.AuthMethod == "hmac_sha256" && recv.SecretKey != "" {
		sig := r.Header.Get("X-Cascata-Signature")
		if sig == "" { sig = r.Header.Get("X-Hub-Signature-256") }
		
		if sig == "" {
			return http.StatusUnauthorized, "Missing signature"
		}

		mac := hmac.New(sha256.New, []byte(recv.SecretKey))
		mac.Write(body)
		expected := hex.EncodeToString(mac.Sum(nil))

		if sig != expected {
			return http.StatusUnauthorized, "Invalid signature"
		}
	}

	// 4. Dispatch (Mocking dispatch for now as Automations are Phase 10)
	// In Go, we'll use a channel or a worker pool for 'dispatchAsyncTrigger'.
	duration := int(time.Since(start).Milliseconds())
	s.LogRun(ctx, projectSlug, "WEBHOOK", recv.ID, "success", duration, string(body), nil, "")

	return http.StatusOK, "Accepted"
}

// InvokeOutbound sends a request to a remote URL with SSRF protection and retry logic.
func (s *Service) InvokeOutbound(ctx context.Context, url string, method string, headers map[string]string, body io.Reader) (*http.Response, error) {
	// 1. SSRF Check
	if err := utils.IsSSRFSafe(url); err != nil {
		return nil, err
	}

	// 2. Prepare Request
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	// 3. Execution with standard retry
	var resp *http.Response
	for i := 0; i < 3; i++ {
		resp, err = client.Do(req)
		if err == nil && resp.StatusCode < 500 {
			break
		}
		time.Sleep(time.Duration(i+1) * time.Second)
	}

	return resp, err
}
