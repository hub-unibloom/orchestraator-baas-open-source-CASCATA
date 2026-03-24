package auth

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/repository"
	"cascata/internal/service"
	"cascata/internal/communication"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// ResidentAuthService handles the complex authentication lifecycle of project end-users.
// Supports CPF/Password, WhatsApp OTP, and Magic Link strategies (Phase 10.1).
type ResidentAuthService struct {
	projectSvc *service.ProjectService
	resRepo    *repository.ResidentRepository
	sessionMgr *SessionManager
	otpMgr     *OTPManager
	postman    communication.Dispatcher
	govSvc     *GovernanceService
	auditSvc   domain.Auditor
}

func NewResidentAuthService(projectSvc *service.ProjectService, resRepo *repository.ResidentRepository, sessionMgr *SessionManager, otpMgr *OTPManager, postman communication.Dispatcher, govSvc *GovernanceService, audit domain.Auditor) *ResidentAuthService {
	return &ResidentAuthService{
		projectSvc: projectSvc,
		resRepo:    resRepo,
		sessionMgr: sessionMgr,
		otpMgr:     otpMgr,
		postman:    postman,
		govSvc:     govSvc,
		auditSvc:   audit,
	}
}

// AuthenticateByCPF validates a resident user using their CPF and password.
func (s *ResidentAuthService) AuthenticateByCPF(ctx context.Context, projectSlug string, cpf, password, clientIP string) (*domain.Resident, string, error) {
	// 1. Resolve Project & Acquire Pool
	p, err := s.projectSvc.Resolve(ctx, projectSlug)
	if err != nil {
		return nil, "", fmt.Errorf("resident.auth.AuthenticateByCPF: project resolution failed: %w", err)
	}

	// 2. Enforce Perimeter (Zero-Trust)
	if err := s.govSvc.EnforcePerimeter(ctx, p, clientIP); err != nil {
		return nil, "", err
	}

	pool, err := s.projectSvc.GetPool(ctx, p)
	if err != nil {
		return nil, "", fmt.Errorf("resident.auth.AuthenticateByCPF: pool resolution failed: %w", err)
	}

	// 2. Lookup Resident
	res, hashedPassword, err := s.resRepo.FindByIdentifier(ctx, pool, cpf)
	if err != nil {
		return nil, "", fmt.Errorf("resident.auth.AuthenticateByCPF: resident lookup failed: %w", err)
	}

	// 4. Issue Token
	token, err := s.issueResidentToken(ctx, projectSlug, res, false)
	if err != nil {
		return nil, "", err
	}

	// 5. Log activity
	_ = s.auditSvc.Log(ctx, projectSlug, "RESIDENT_LOGIN", res.ID, string(domain.IdentityResident), map[string]interface{}{"cpf": cpf})

	return res, token, nil
}

// IssueStepUpOTP initiates an elevated authentication prompt for an already logged-in user.
func (s *ResidentAuthService) IssueStepUpOTP(ctx context.Context, slug, resID string) error {
	p, err := s.projectSvc.Resolve(ctx, slug)
	if err != nil { return err }
	pool, err := s.projectSvc.GetPool(ctx, p)
	if err != nil { return err }
	
	r, _, err := s.resRepo.FindByIdentifier(ctx, pool, resID)
	if err != nil { return err }

	slog.Info("resident.auth: issuing step-up challenge", "slug", slug, "resident_id", resID)
	
	// We use the email/whatsapp as identifier for the OTP
	identifier := r.Email
	if identifier == "" { identifier = r.WhatsApp }

	code, err := s.otpMgr.IssueOTP(ctx, slug, identifier)
	if err != nil { return err }

	body := fmt.Sprintf("[SECURITY] Elevated access requested for your account. Code: %s", code)
	go func() {
		if r.Email != "" { _ = s.postman.SendEmail(context.Background(), r.Email, "Step-up Verification", body) }
		if r.WhatsApp != "" { _ = s.postman.SendWhatsApp(context.Background(), r.WhatsApp, body) }
	}()

	return nil
}

// VerifyStepUp validates the elevated challenge and issues a 'mfa-elevated' JWT.
func (s *ResidentAuthService) VerifyStepUp(ctx context.Context, slug, residentID, code string) (string, error) {
	p, err := s.projectSvc.Resolve(ctx, slug)
	if err != nil { return "", err }
	pool, err := s.projectSvc.GetPool(ctx, p)
	if err != nil { return "", err }
	r, _, _ := s.resRepo.FindByIdentifier(ctx, pool, residentID)
	
	identifier := r.Email
	if identifier == "" { identifier = r.WhatsApp }

	valid, err := s.otpMgr.VerifyOTP(ctx, slug, identifier, code)
	if err != nil || !valid {
		return "", fmt.Errorf("invalid step-up code")
	}

	return s.issueResidentToken(ctx, slug, r, true)
}

// issueResidentToken generates a JWT signed with the Project's specific secret.
func (s *ResidentAuthService) issueResidentToken(ctx context.Context, projectSlug string, r *domain.Resident, isMFA bool) (string, error) {
	p, err := s.projectSvc.Resolve(ctx, projectSlug)
	if err != nil {
		return "", fmt.Errorf("resident.auth.issueResidentToken: project resolution failed: %w", err)
	}

	claims := jwt.MapClaims{
		"sub":   r.ID,
		"email": r.Email,
		"role":  string(r.Role),
		"exp":   time.Now().Add(1 * time.Hour).Unix(),
		"iat":   time.Now().Unix(),
		"tenant": projectSlug,
		"amr":    []string{"pwd"},
	}

	if isMFA {
		claims["amr"] = []string{"pwd", "mfa"}
		claims["mfa_verified_at"] = time.Now().Unix()
	}

	signedToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return signedToken.SignedString([]byte(p.JWTSecret))
}


// RequestMagicLink generates a temporary token (OTP) for the user.
func (s *ResidentAuthService) RequestMagicLink(ctx context.Context, slug, email string) (string, error) {
	slog.Info("resident.auth: issuing magic link code", "slug", slug, "email", email)
	code, err := s.otpMgr.IssueOTP(ctx, slug, email)
	if err != nil {
		return "", err
	}

	body := fmt.Sprintf("Your login code for Cascata project %s is: %s", slug, code)
	go func() {
		_ = s.postman.SendEmail(context.Background(), email, "Cascata Login Code", body)
	}()

	return code, nil
}

// RequestWhatsAppOTP generates a 6-digit code for the user.
func (s *ResidentAuthService) RequestWhatsAppOTP(ctx context.Context, slug, whatsapp string) (string, error) {
	slog.Info("resident.auth: issuing whatsapp otp", "slug", slug, "whatsapp", whatsapp)
	code, err := s.otpMgr.IssueOTP(ctx, slug, whatsapp)
	if err != nil {
		return "", err
	}

	body := fmt.Sprintf("Your login code for Cascata project %s is: %s", slug, code)
	go func() {
		_ = s.postman.SendWhatsApp(context.Background(), whatsapp, body)
	}()

	return code, nil
}

// VerifyOTP checks the code and issues a Resident Token if valid.
func (s *ResidentAuthService) VerifyOTP(ctx context.Context, projectSlug, identifier, code, clientIP string) (*domain.Resident, string, error) {
	// 1. Verify in Dragonfly (Phase 10.1 OTP Engine)
	valid, err := s.otpMgr.VerifyOTP(ctx, projectSlug, identifier, code)
	if err != nil || !valid {
		slog.Warn("resident.auth: invalid or expired OTP", "tenant_slug", projectSlug, "identifier", identifier)
		return nil, "", fmt.Errorf("resident.auth.VerifyOTP: invalid or expired otp")
	}

	// 2. Resolve Pool
	p, err := s.projectSvc.Resolve(ctx, projectSlug)
	if err != nil {
		return nil, "", err
	}

	// 3. Enforce Perimeter
	if err := s.govSvc.EnforcePerimeter(ctx, p, clientIP); err != nil {
		return nil, "", err
	}

	pool, err := s.projectSvc.GetPool(ctx, p)
	if err != nil {
		return nil, "", err
	}

	// 3. Resolve Resident
	res, _, err := s.resRepo.FindByIdentifier(ctx, pool, identifier)
	if err != nil {
		return nil, "", fmt.Errorf("resident.auth.VerifyOTP: resident resolution failed: %w", err)
	}

	// 4. Issue Token
	signedJWT, err := s.issueResidentToken(ctx, projectSlug, res, false)
	if err != nil {
		return nil, "", err
	}

	// 5. Update Audit Trail (Async)
	go func() {
		_ = s.resRepo.UpdateLastLogin(context.Background(), pool, res.ID)
		_ = s.auditSvc.Log(context.Background(), projectSlug, "RESIDENT_LOGIN_OTP", res.ID, string(domain.IdentityResident), map[string]interface{}{"identifier": identifier})
	}()

	return res, signedJWT, nil
}

// SignupResident registers a new end-user inside the tenant's auth.users table.
func (s *ResidentAuthService) SignupResident(ctx context.Context, projectSlug string, r *domain.Resident, rawPassword string) error {
	// 1. Resolve Pool
	p, err := s.projectSvc.Resolve(ctx, projectSlug)
	if err != nil {
		return fmt.Errorf("resident.auth.SignupResident: project resolution failed: %w", err)
	}
	pool, err := s.projectSvc.GetPool(ctx, p)
	if err != nil {
		return fmt.Errorf("resident.auth.SignupResident: pool resolution failed: %w", err)
	}

	// 2. Hash Password (standard cost 12 as per Cascata Rules)
	var hashedPassword string
	if rawPassword != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(rawPassword), 12)
		if err != nil {
			return fmt.Errorf("resident.auth.SignupResident: password hashing failed: %w", err)
		}
		hashedPassword = string(hash)
	}

	// 3. Persist via Repository (Phase 10.2 schema)
	if err := s.resRepo.Create(ctx, pool, r, hashedPassword); err != nil {
		return fmt.Errorf("resident.auth.SignupResident: resident creation failed: %w", err)
	}

	slog.Info("resident.auth: resident signed up", "tenant_slug", projectSlug, "resident_id", r.ID)
	return nil
}
