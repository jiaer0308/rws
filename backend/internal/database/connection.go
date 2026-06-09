package database

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Pool represents the global database connection pool
var Pool *pgxpool.Pool

// ConnectDB initializes the pgx connection pool and verifies the connection
func ConnectDB(ctx context.Context) (*pgxpool.Pool, error) {
	connStr := os.Getenv("DATABASE_URL")
	if connStr == "" {
		// Fallback to default local development connection
		connStr = "postgres://postgres:Panda24685l$@localhost:5432/rwsDB?sslmode=disable"
	}

	config, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		return nil, fmt.Errorf("unable to parse connection string: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Verify database connection is alive
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}

	Pool = pool
	return pool, nil
}
