package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"bluecollarjob/internal/config"
	"bluecollarjob/internal/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const migrationVersion = "000001_init_schema"

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: go run ./cmd/migrate [up|down|status]")
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	db, err := database.NewPostgresPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connect postgres: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		log.Fatalf("ensure schema_migrations: %v", err)
	}

	switch os.Args[1] {
	case "up":
		runMigration(ctx, db, true)
	case "down":
		runMigration(ctx, db, false)
	case "status":
		printStatus(ctx, db)
	default:
		log.Fatalf("unknown migration command %q", os.Args[1])
	}
}

type execer interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func runMigration(ctx context.Context, db execer, up bool) {
	applied := isApplied(ctx, db)
	if up && applied {
		fmt.Printf("%s already applied\n", migrationVersion)
		return
	}
	if !up && !applied {
		fmt.Printf("%s is not applied\n", migrationVersion)
		return
	}

	suffix := "up"
	if !up {
		suffix = "down"
	}
	sqlText, err := readMigrationSQL(suffix)
	if err != nil {
		log.Fatalf("read migration: %v", err)
	}
	if _, err := db.Exec(ctx, sqlText); err != nil {
		log.Fatalf("execute migration %s: %v", suffix, err)
	}
	if up {
		if _, err := db.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1) ON CONFLICT (version) DO NOTHING`, migrationVersion); err != nil {
			log.Fatalf("record migration: %v", err)
		}
		fmt.Printf("%s applied\n", migrationVersion)
		return
	}
	if _, err := db.Exec(ctx, `DELETE FROM schema_migrations WHERE version = $1`, migrationVersion); err != nil {
		log.Fatalf("delete migration record: %v", err)
	}
	fmt.Printf("%s rolled back\n", migrationVersion)
}

func printStatus(ctx context.Context, db execer) {
	status := "pending"
	if isApplied(ctx, db) {
		status = "applied"
	}
	fmt.Printf("%s %s\n", migrationVersion, status)
}

func isApplied(ctx context.Context, db execer) bool {
	var exists bool
	err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE version = $1)`, migrationVersion).Scan(&exists)
	return err == nil && exists
}

func readMigrationSQL(suffix string) (string, error) {
	candidates := []string{
		filepath.Join("migrations", migrationVersion+"."+suffix+".sql"),
		migrationVersion + "." + suffix + ".sql",
	}
	for _, candidate := range candidates {
		data, err := os.ReadFile(candidate)
		if err == nil {
			return string(data), nil
		}
	}
	return "", fmt.Errorf("could not find %s migration SQL", suffix)
}
