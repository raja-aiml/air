package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

// NewPool creates a pgx connection pool with sane defaults.
func NewPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}
	cfg.MaxConns = 10
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 55 * time.Minute
	cfg.MaxConnIdleTime = 5 * time.Minute
	cfg.HealthCheckPeriod = 30 * time.Second
	return pgxpool.NewWithConfig(ctx, cfg)
}

// RunMigrations applies embedded SQL migrations in order.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	migrations, err := loadMigrations()
	if err != nil {
		return err
	}
	if len(migrations) == 0 {
		return nil
	}

	if err := ensureSchemaTable(ctx, pool); err != nil {
		return err
	}

	applied, err := appliedVersions(ctx, pool)
	if err != nil {
		return err
	}
	appliedSet := make(map[int]struct{}, len(applied))
	for _, v := range applied {
		appliedSet[v] = struct{}{}
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin migration tx: %w", err)
	}
	defer tx.Rollback(ctx)

	for _, m := range migrations {
		if _, seen := appliedSet[m.Version]; seen {
			continue
		}
		if _, err := tx.Exec(ctx, m.Content); err != nil {
			return fmt.Errorf("apply migration %d: %w", m.Version, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, m.Version); err != nil {
			return fmt.Errorf("record migration %d: %w", m.Version, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migrations: %w", err)
	}
	return nil
}

type migration struct {
	Version int
	Name    string
	Content string
}

var versionPattern = regexp.MustCompile(`^(\d+)_?.*\.sql$`)

func loadMigrations() ([]migration, error) {
	files, err := fs.Glob(migrationFiles, "migrations/*.sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	var migrations []migration
	for _, path := range files {
		base := filepath.Base(path)
		m := versionPattern.FindStringSubmatch(base)
		if len(m) < 2 {
			return nil, fmt.Errorf("invalid migration filename: %s", base)
		}
		ver, err := strconv.Atoi(m[1])
		if err != nil {
			return nil, fmt.Errorf("parse version from %s: %w", base, err)
		}
		body, err := migrationFiles.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", base, err)
		}
		migrations = append(migrations, migration{
			Version: ver,
			Name:    strings.TrimSuffix(base, ".sql"),
			Content: string(body),
		})
	}
	sort.Slice(migrations, func(i, j int) bool { return migrations[i].Version < migrations[j].Version })
	return migrations, nil
}

func ensureSchemaTable(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	return nil
}

func appliedVersions(ctx context.Context, pool *pgxpool.Pool) ([]int, error) {
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("select schema_migrations: %w", err)
	}
	defer rows.Close()
	var versions []int
	for rows.Next() {
		var v int
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("scan schema_migrations: %w", err)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return versions, nil
}

// Ping verifies connectivity to the database.
func Ping(ctx context.Context, pool *pgxpool.Pool) error {
	var one int
	return pool.QueryRow(ctx, "SELECT 1").Scan(&one)
}
