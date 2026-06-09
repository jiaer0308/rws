package database

import (
	"context"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrations runs all SQL migration files in the database/migrations directory.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	if pool == nil {
		return fmt.Errorf("database pool is not initialized")
	}

	// Try a few possible paths for migrations directory depending on execution directory
	possiblePaths := []string{
		"../database/migrations",
		"database/migrations",
		"./database/migrations",
		"../../database/migrations",
	}

	var migrationDir string
	for _, p := range possiblePaths {
		if _, err := os.Stat(p); err == nil {
			migrationDir = p
			break
		}
	}

	if migrationDir == "" {
		return fmt.Errorf("could not find migrations directory (checked: %v)", possiblePaths)
	}

	log.Printf("Running database migrations from: %s", migrationDir)

	files, err := ioutil.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var sqlFiles []string
	for _, file := range files {
		if !file.IsDir() && strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, filepath.Join(migrationDir, file.Name()))
		}
	}

	// Ensure migrations are run in order
	sort.Strings(sqlFiles)

	// Create migrations tracking table if it doesn't exist
	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			run_at TIMESTAMPTZ DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("failed to create schema_migrations table: %w", err)
	}

	for _, sqlFile := range sqlFiles {
		version := filepath.Base(sqlFile)

		// Check if migration has already been run
		var exists bool
		err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&exists)
		if err != nil {
			return fmt.Errorf("failed to check migration state for %s: %w", version, err)
		}

		if exists {
			log.Printf("Migration %s already applied, skipping.", version)
			continue
		}

		log.Printf("Applying migration: %s", version)
		content, err := ioutil.ReadFile(sqlFile)
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", sqlFile, err)
		}

		// Run migration content in a transaction
		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer tx.Rollback(ctx)

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("failed to execute migration %s: %w", version, err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO schema_migrations (version) VALUES ($1)", version); err != nil {
			return fmt.Errorf("failed to log migration %s: %w", version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}

		log.Printf("Migration %s applied successfully.", version)
	}

	return nil
}
