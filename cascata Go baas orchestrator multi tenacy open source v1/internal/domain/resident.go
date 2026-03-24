package domain

import (
	"time"
)

// ResidentRole represents the permission level of an end-user within a Tenant.
type ResidentRole string

const (
	RoleResident      ResidentRole = "authenticated"
	RoleAdmin         ResidentRole = "admin"
	RoleServiceAccount ResidentRole = "service_account"
)

// AuthStrategy defines the method used for authentication.
type AuthStrategy string

const (
	StrategyPassword    AuthStrategy = "password"
	StrategyOTPWhatsApp AuthStrategy = "whatsapp_otp"
	StrategyMagicLink   AuthStrategy = "magic_link"
)

// Resident represents a Resident (end-user) of a specific Tenant project.
// Part of Phase 10.1: Resident Gateway.
type Resident struct {
	ID             string                 `json:"id"`
	ProjectSlug    string                 `json:"project_slug"`
	Email          string                 `json:"email,omitempty"`
	CPF            string                 `json:"cpf,omitempty"`
	WhatsApp       string                 `json:"whatsapp,omitempty"`
	PasswordHash   string                 `json:"-"`
	Role           ResidentRole           `json:"role"`
	IsEmailVerified bool                   `json:"is_email_verified"`
	IsPhoneVerified bool                   `json:"is_phone_verified"`
	Metadata       map[string]interface{} `json:"metadata"`
	LastLogin      *time.Time             `json:"last_login,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// AuthPayload defines the generic structure for login attempts across strategies.
type AuthPayload struct {
	Identifier string       `json:"identifier"` // Email, CPF, or WhatsApp
	Password   string       `json:"password,omitempty"`
	Code       string       `json:"code,omitempty"` // OTP or Magic Link Token
	Strategy   AuthStrategy `json:"strategy"`
}
