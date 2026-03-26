package main

import (
	"context"
	"fmt"
	"os"

	"cascata/internal/crypto"
	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/repository"
	"cascata/internal/service"
	"github.com/jackc/pgx/v5"
)

// One-off CLI helper to hash passwords and provision the Worner
// during the install.sh phase. This avoids passing cleartext 
// passwords as SQL arguments to psql.
func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: worner-provision <email> <password> <mfa_enabled>")
		os.Exit(1)
	}

	email := os.Args[1]
	password := os.Args[2]
	mfaEnabled := os.Args[3] == "true"

	// 1. Generate secure Argon2id Hash (Phase 10 Hardening)
	hash, err := crypto.HashPassword(password)
	if err != nil {
		fmt.Printf("FAIL_HASH: %v\n", err)
		os.Exit(1)
	}

	// 2. Connect to Metadata Database
	dbURL := os.Getenv("DB_URL")
	if dbURL == "" {
		fmt.Println("FAIL_DB_CONFIG: DB_URL is required")
		os.Exit(1)
	}
	
	db, err := database.Connect(context.Background(), dbURL)
	if err != nil {
		fmt.Printf("FAIL_DB_CONNECT: %v\n", err)
		os.Exit(1)
	}
	defer db.Close()

	// 3. Definitions and Provisioning logic (Sovereign Context)
	repo := repository.NewMemberRepository(db)
	member := &domain.Member{
		Email:        email,
		PasswordHash: hash,
		Role:         domain.RoleWorner,
		Type:         domain.IdentityMember,
		MFAEnabled:   mfaEnabled,
	}

	claims := database.UserClaims{Role: "service_role"}
	err = db.WithRLS(context.Background(), claims, "cascata", false, func(tx pgx.Tx) error {
		if err := repo.Create(context.Background(), tx, member); err != nil {
			return err
		}

		// 4. Initial Audit Entry via Unified Ledger
		audit := service.NewAuditService(db)
		return audit.Log(context.Background(), "cascata", "GENESIS_PROVISION_WORNER", member.ID, string(domain.IdentityMember), map[string]interface{}{
			"email":  email,
			"source": "install.sh_go_helper",
		})
	})

	if err != nil {
		fmt.Printf("FAIL_DB_GENESIS: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("SUCCESS_ID:%s\n", member.ID)
}
