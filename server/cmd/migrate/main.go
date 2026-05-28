package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/leo/iop/server/internal/config"
)

// Minimal in-house migrator (M1). M2/M3 may swap in golang-migrate if needs grow.
//
// File naming: NNNNNN_<name>.up.sql / .down.sql (e.g. 000001_init.up.sql).
// Tracking: public.migration_history (created by 000001_init).
//
// Commands:
//   migrate up       — apply all pending migrations
//   migrate status   — list applied + pending

func main() {
	dir := flag.String("dir", "./migrations/public", "migrations directory")
	flag.Parse()

	cmd := flag.Arg(0)
	if cmd == "" {
		cmd = "up"
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DB.DSN)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	files, err := loadMigrations(*dir)
	if err != nil {
		log.Fatalf("load migrations: %v", err)
	}

	switch cmd {
	case "up":
		applied, _ := loadApplied(ctx, conn)
		count := 0
		for _, m := range files {
			if applied[m.id] {
				continue
			}
			log.Printf("applying %s ...", m.id)
			if err := apply(ctx, conn, m); err != nil {
				log.Fatalf("apply %s: %v", m.id, err)
			}
			count++
		}
		log.Printf("migrated %d new migrations", count)
	case "status":
		applied, _ := loadApplied(ctx, conn)
		for _, m := range files {
			tag := "PENDING"
			if applied[m.id] {
				tag = "applied"
			}
			fmt.Printf("%-10s  %s\n", tag, m.id)
		}
	default:
		log.Fatalf("unknown command: %s (use: up | status)", cmd)
	}
}

type migration struct {
	id   string // e.g. "000001_init"
	path string
	sql  string
	sum  string
}

func loadMigrations(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read dir %s: %w", dir, err)
	}
	var out []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".up.sql")
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		h := sha256.Sum256(raw)
		out = append(out, migration{
			id:   id,
			path: filepath.Join(dir, e.Name()),
			sql:  string(raw),
			sum:  hex.EncodeToString(h[:]),
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out, nil
}

func loadApplied(ctx context.Context, conn *pgx.Conn) (map[string]bool, error) {
	out := map[string]bool{}
	rows, err := conn.Query(ctx,
		"SELECT migration_id FROM public.migration_history WHERE scope = 'public'")
	if err != nil {
		// Table may not exist on first run — treat as empty.
		return out, nil
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return out, err
		}
		out[id] = true
	}
	return out, nil
}

func apply(ctx context.Context, conn *pgx.Conn, m migration) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, m.sql); err != nil {
		return err
	}
	// Record in history (skip for 000001_init which inserts itself).
	if m.id != "000001_init" {
		_, err = tx.Exec(ctx,
			"INSERT INTO public.migration_history (id, scope, migration_id, applied_at, checksum) VALUES ($1, 'public', $2, $3, $4) ON CONFLICT DO NOTHING",
			uuid.New(), m.id, time.Now().UTC(), m.sum)
		if err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
