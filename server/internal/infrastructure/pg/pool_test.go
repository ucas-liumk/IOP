package pg

import (
	"context"
	"os"
	"testing"
	"time"
)

func dsnFromEnv(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("IOP_TEST_DB_DSN")
	if dsn == "" {
		dsn = "postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable"
	}
	return dsn
}

func TestNewPool_Connects(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := NewPool(ctx, Config{DSN: dsnFromEnv(t), MaxConns: 4})
	if err != nil {
		t.Skipf("PG not reachable (start docker-compose first): %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestNewPool_InvalidDSN(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := NewPool(ctx, Config{DSN: "postgres://nope:nope@127.0.0.1:1/nope", MaxConns: 1})
	if err == nil {
		t.Fatalf("expected connection error")
	}
}
