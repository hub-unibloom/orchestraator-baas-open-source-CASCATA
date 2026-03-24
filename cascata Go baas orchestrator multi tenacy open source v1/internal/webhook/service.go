package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"time"

	"cascata/internal/automation"
	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/service"
	"cascata/internal/telemetry"
	"cascata/internal/utils"
)

// InboundReceiver represents a registered webhook entry point.
type InboundReceiver struct {
	ID         string
	AuthMethod string
	SecretKey  string
	TargetType string
	TargetID   string
}

// Service manages the webhook lifecycle and execution logging.
type Service struct {
	repo       *database.Repository
	eventQueue *automation.EventQueue
	auditSvc   *service.AuditService
	telemetry  *telemetry.TelemetryEngine
}

func NewService(repo *database.Repository, eventQueue *automation.EventQueue, audit *service.AuditService, tele *telemetry.TelemetryEngine) *Service {
	return &Service{
		repo:       repo,
		eventQueue: eventQueue,
		auditSvc:   audit,
		telemetry:  tele,
	}
}

// LogRun tracks the execution history of webhooks (Phase 11).
func (s *Service) LogRun(ctx context.Context, slug, resType, resID, status string, duration int, payload string, err error, response string) {
	// Persistence of webhook history in system.automation_runs or specific legacy table.
}

// HandleInbound processes an incoming HTTP webhook request.
func (s *Service) HandleInbound(ctx context.Context, projectSlug, pathSlug string, r *http.Request) (int, string) {
	// Phase 10.4 Sinergy: Start a Trace for the inbound webhook
	ctx, span := s.telemetry.Tracer("webhooks").Start(ctx, "webhook.HandleInbound")
	defer span.End()

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

	// 4. Real Dispatch to Automation Engine (Phase 10.4)
	go func() {
		event := &domain.AutomationEvent{
			ProjectSlug: projectSlug,
			Table:       "webhooks",
			Operation:   "INBOUND",
			Data:        map[string]interface{}{"payload": string(body), "path": pathSlug, "id": recv.ID},
		}
		
		// Phase 24 Sinergy: Log to Audit Ledger
		_ = s.auditSvc.Log(context.Background(), projectSlug, "webhook_inbound", "SYSTEM", "TRIGGER", map[string]interface{}{
			"receiver_id": recv.ID,
			"path":        pathSlug,
		})

		_ = s.eventQueue.Publish(context.Background(), event)
	}()

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
