package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"strings"

	"github.com/kadebhug/seatd_v2/internal/app"
	"github.com/kadebhug/seatd_v2/internal/store"
	"github.com/kadebhug/seatd_v2/internal/store/db"
)

func main() {
	email := flag.String("email", "", "email address to grant platform admin access")
	addedBy := flag.String("added-by", "seed:cli", "actor reference recorded as the grant creator")
	role := flag.String("role", "platform_admin", "platform role: platform_admin or support")
	flag.Parse()

	normalizedEmail, err := normalizeEmail(*email)
	if err != nil {
		slog.Error("email is required and must be valid", "error", err)
		os.Exit(2)
	}
	normalizedRole := strings.TrimSpace(*role)
	if normalizedRole != "platform_admin" && normalizedRole != "support" {
		slog.Error("role must be platform_admin or support")
		os.Exit(2)
	}
	actorRef := strings.TrimSpace(*addedBy)
	if actorRef == "" {
		slog.Error("added-by is required")
		os.Exit(2)
	}

	ctx := context.Background()
	cfg, err := app.LoadConfig("api", 8080, app.BuildInfo{})
	if err != nil {
		slog.Error("configuration invalid", "error", err)
		os.Exit(1)
	}
	if cfg.DatabaseURL == "" {
		slog.Error("database url is required", "variable", "SEATD_DATABASE_URL")
		os.Exit(1)
	}
	pool, err := store.OpenPool(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	tx, err := pool.Begin(ctx)
	if err != nil {
		slog.Error("beginning transaction failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()
	q := db.New(tx)
	if err := q.SetPlatformAdminContext(ctx); err != nil {
		slog.Error("setting platform context failed", "error", err)
		os.Exit(1)
	}
	grant, err := q.CreatePlatformAdminGrant(ctx, db.CreatePlatformAdminGrantParams{
		Btrim:             normalizedEmail,
		Role:              normalizedRole,
		InvitedByActorRef: actorRef,
	})
	if err != nil {
		slog.Error("creating platform admin grant failed", "error", err)
		os.Exit(1)
	}
	if err := tx.Commit(ctx); err != nil {
		slog.Error("committing platform admin grant failed", "error", err)
		os.Exit(1)
	}
	fmt.Printf("seeded platform grant %s for %s as %s\n", grant.ID, grant.Email, grant.Role)
}

func normalizeEmail(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("empty email")
	}
	address, err := mail.ParseAddress(value)
	if err != nil || strings.TrimSpace(address.Address) == "" {
		return "", fmt.Errorf("invalid email")
	}
	return strings.ToLower(address.Address), nil
}
