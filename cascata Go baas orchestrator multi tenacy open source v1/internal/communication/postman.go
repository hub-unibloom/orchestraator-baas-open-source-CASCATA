package communication

import (
	"context"
	"fmt"
	"log/slog"
	"net/smtp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// Dispatcher defines the interface for sending transactional communications.
type Dispatcher interface {
	SendEmail(ctx context.Context, to, subject, body string) error
	SendWhatsApp(ctx context.Context, to, message string) error
	SendPush(ctx context.Context, token, title, body string, data map[string]string) error
}

// TrinityPostman is the central hub for all transactional messages (Phase 10.3).
// Integrated with Vault for secure credential retrieval.
type TrinityPostman struct {
	smtpHost     string
	smtpPort     string
	smtpUser     string
	smtpPassword string
	fromEmail    string
}

func NewTrinityPostman(host, port, user, password, from string) *TrinityPostman {
	return &TrinityPostman{
		smtpHost:     host,
		smtpPort:     port,
		smtpUser:     user,
		smtpPassword: password,
		fromEmail:    from,
	}
}

// SendEmail dispatches a real SMTP message using Cascata's secure infrastructure.
func (p *TrinityPostman) SendEmail(ctx context.Context, to, subject, body string) error {
	tr := otel.Tracer("communication")
	ctx, span := tr.Start(ctx, "SendEmail", trace.WithAttributes(
		attribute.String("messaging.system", "smtp"),
		attribute.String("messaging.destination", to),
	))
	defer span.End()

	auth := smtp.PlainAuth("", p.smtpUser, p.smtpPassword, p.smtpHost)
	
	msg := []byte(fmt.Sprintf("To: %s\r\nSubject: %s\r\n\r\n%s\r\n", to, subject, body))
	addr := fmt.Sprintf("%s:%s", p.smtpHost, p.smtpPort)

	// In Phase 10.3, we use a goroutine for non-blocking dispatch in production.
	// But the service call itself returns error if it fails basic validation.
	err := smtp.SendMail(addr, auth, p.fromEmail, []string{to}, msg)
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("postman.SendEmail: SMTP dispatch failed: %w", err)
	}

	slog.Info("postman: email sent successfully", "to", to, "subject", subject)
	return nil
}

// SendWhatsApp dispatches a message via a high-performance WhatsApp API provider.
func (p *TrinityPostman) SendWhatsApp(ctx context.Context, to, message string) error {
	// REAL API Integration (Twilio/Meta) pending Vault credentials in Phase 10.3
	return fmt.Errorf("postman.SendWhatsApp: not implemented")
}

// SendPush dispatches a mobile notification via FCM (Firebase Cloud Messaging).
func (p *TrinityPostman) SendPush(ctx context.Context, token, title, body string, data map[string]string) error {
	// Real Push Integration (Firebase) pending service account JSON in Phase 10.4
	return fmt.Errorf("postman.SendPush: not implemented")
}
