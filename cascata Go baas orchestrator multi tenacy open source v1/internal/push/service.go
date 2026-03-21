package push

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"cascata/internal/database"
)

// PulseManager handles automated push notifications based on DB events.
type PulseManager struct {
	repo *database.Repository
}

func NewPulseManager(repo *database.Repository) *PulseManager {
	return &PulseManager{repo: repo}
}

// NotificationRule defines when and how to send a push.
type NotificationRule struct {
	TriggerTable    string `json:"trigger_table"`
	TriggerEvent    string `json:"trigger_event"`
	TitleTemplate   string `json:"title_template"`
	BodyTemplate    string `json:"body_template"`
	RecipientColumn string `json:"recipient_column"`
}

// ProcessEventTrigger evaluates if a DB event should fire a Neural Pulse.
func (p *PulseManager) ProcessEventTrigger(ctx context.Context, slug string, event interface{}) {
	// 1. Fetch Rules from metadata
	// 2. Resolve templates {{...}}
	// 3. Send to FCM v1 API
}

func (p *PulseManager) sendFCM(ctx context.Context, token, title, body string) error {
	// FCM v1 API Implementation
	// Endpoint: https://fcm.googleapis.com/v1/projects/{fcm_project_id}/messages:send
	return nil
}

// Helper to resolve {{template}}
func resolve(template string, record map[string]interface{}) string {
	res := template
	for k, v := range record {
		tag := fmt.Sprintf("{{%s}}", k)
		res = strings.ReplaceAll(res, tag, fmt.Sprintf("%v", v))
	}
	return res
}
