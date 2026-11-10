// admin-set: utilitário CLI para criar ou atualizar o admin principal.
// Lê ADMIN_EMAIL e ADMIN_PASSWORD do ambiente e faz upsert na tabela admin_users.
//
// Uso:
//
//	set -a && source ../.env && set +a && go run ./cmd/admin-set
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/renankeny/conversion-tracker/backend/internal/auth"
	"github.com/renankeny/conversion-tracker/backend/internal/config"
	"github.com/renankeny/conversion-tracker/backend/internal/util"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	config.MustSetTimezone()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("config", "err", err)
		os.Exit(1)
	}

	if cfg.AdminEmail == "" || cfg.AdminPassword == "" {
		slog.Error("ADMIN_EMAIL e ADMIN_PASSWORD são obrigatórios no .env")
		os.Exit(1)
	}

	if err := util.SetEncryptionKey(cfg.EncryptionKey); err != nil {
		slog.Error("failed to set encryption key", "err", err)
		os.Exit(1)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("db connect", "err", err)
		os.Exit(1)
	}
	defer pool.Close()

	hash, err := auth.HashPassword(cfg.AdminPassword)
	if err != nil {
		slog.Error("hash password", "err", err)
		os.Exit(1)
	}

	// Upsert: se o email já existir, atualiza a senha; senão, insere.
	// Sempre marca como owner com acesso global (não vinculado a account).
	_, err = pool.Exec(ctx, `
		INSERT INTO admin_users (email, password_hash, role, account_id)
		VALUES ($1, $2, 'owner', NULL)
		ON CONFLICT (email) DO UPDATE
		   SET password_hash = EXCLUDED.password_hash,
		       role = 'owner',
		       updated_at = NOW()
	`, cfg.AdminEmail, hash)
	if err != nil {
		slog.Error("upsert admin", "err", err)
		os.Exit(1)
	}

	slog.Info("admin atualizado com sucesso", "email", cfg.AdminEmail)
}
