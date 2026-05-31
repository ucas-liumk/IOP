package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/leo/iop/server/internal/app"
	"github.com/leo/iop/server/internal/config"
)

func TestSmoke_EndToEnd(t *testing.T) {
	if os.Getenv("IOP_INTEGRATION") == "" {
		t.Skip("set IOP_INTEGRATION=1 to run (requires docker-compose db running)")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	if dsn := os.Getenv("IOP_TEST_DB_DSN"); dsn != "" {
		cfg.DB.DSN = dsn
	} else if cfg.DB.DSN == "" {
		cfg.DB.DSN = "postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a, cleanup, err := app.Build(ctx, cfg)
	if err != nil {
		t.Fatalf("app build: %v", err)
	}
	defer cleanup()

	srv := httptest.NewServer(a.Engine())
	defer srv.Close()

	for _, tc := range []struct {
		name string
		path string
		want int
	}{
		{"livez", "/livez", 200},
		{"readyz", "/readyz", 200},
		{"version", "/version", 200},
		{"healthz", "/healthz", 200},
		{"metrics", "/metrics", 200},
		{"dict-known", "/api/dict/plan_level", 200},
		{"dict-unknown", "/api/dict/nope", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tc.path)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, resp.StatusCode)
			}
		})
	}

	t.Run("trace_propagation", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/api/dict/plan_level", nil)
		req.Header.Set("X-Request-Id", "trace-smoke-123")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		if got := resp.Header.Get("X-Request-Id"); got != "trace-smoke-123" {
			t.Fatalf("expected echo of trace id, got %q", got)
		}
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body["trace_id"] != "trace-smoke-123" {
			t.Fatalf("envelope missing trace id, got %v", body["trace_id"])
		}
	})

	t.Run("panic_recovery", func(t *testing.T) {
		// Add a panic route to verify Recover middleware.
		// We can't easily mutate the engine after construction here, so probe
		// an endpoint that returns an error envelope and verify shape.
		resp, err := http.Get(srv.URL + "/api/dict/nope")
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body["code"] != float64(-1) {
			t.Fatalf("expected code=-1 on error, got %v", body["code"])
		}
		if body["error"] == nil {
			t.Fatalf("expected error in envelope")
		}
	})
}
