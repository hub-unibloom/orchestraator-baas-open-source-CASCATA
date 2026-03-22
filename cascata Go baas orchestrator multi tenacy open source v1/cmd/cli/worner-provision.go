package main

import (
	"context"
	"fmt"
	"os"

	"cascata/internal/auth"
	"cascata/internal/database"
	"cascata/internal/domain"
	"cascata/internal/repository"
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

	// 1. Generate secure Bcrypt Hash Cost 12
	hash, err := auth.HashPassword(password)
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

	// 3. Insert and Audit Genesis
	repo := repository.NewMemberRepository(db)
	
	member := &domain.Member{
		Email:        email,
		PasswordHash: hash,
		Role:         domain.RoleWorner,
		Type:         domain.TypeHuman,
		MFAEnabled:   mfaEnabled,
	}

	err = repo.Create(context.Background(), member)
	if err != nil {
		fmt.Printf("FAIL_DB_INSERT: %v\n", err)
		os.Exit(1)
	}

	// Initial Audit Entry
	_ = repo.LogActivity(context.Background(), &domain.AuditLog{
		MemberID:   member.ID,
		MemberType: domain.TypeHuman,
		Action:     "GENESIS_PROVISION_WORNER",
		EntityType: "system.members",
		EntityID:   member.ID,
		Metadata:   map[string]interface{}{"email": email, "source": "install.sh_go_helper"},
	})

	fmt.Printf("SUCCESS_ID:%s\n", member.ID)
}
