package logger

import (
	"io"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config controls log level + output format.
type Config struct {
	Level  string // "debug" | "info" | "warn" | "error"
	Format string // "json" (prod) | "console" (dev)
}

// New returns a configured *zap.Logger writing to stderr.
func New(cfg Config) (*zap.Logger, error) {
	lvl, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		lvl = zapcore.InfoLevel
	}
	zcfg := zap.NewProductionConfig()
	if cfg.Format == "console" {
		zcfg = zap.NewDevelopmentConfig()
	}
	zcfg.Level = zap.NewAtomicLevelAt(lvl)
	zcfg.DisableStacktrace = false
	return zcfg.Build()
}

// newTestable lets tests inject a buffer writer.
func newTestable(w io.Writer, lvl zapcore.Level) *zap.Logger {
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(w), lvl)
	return zap.New(core)
}

// sensitiveKeys are redacted from any structured field map before logging.
var sensitiveKeys = []string{"password", "passwd", "token", "secret", "authorization", "cookie", "api_key"}

// Sanitize walks a map[string]string and replaces values for sensitive keys.
// Extend to map[string]any in M2 when handler logging touches richer payloads.
func Sanitize(v any) any {
	m, ok := v.(map[string]string)
	if !ok {
		return v
	}
	out := make(map[string]string, len(m))
	for k, val := range m {
		lk := strings.ToLower(k)
		redacted := false
		for _, s := range sensitiveKeys {
			if strings.Contains(lk, s) {
				out[k] = "[REDACTED]"
				redacted = true
				break
			}
		}
		if !redacted {
			out[k] = val
		}
	}
	return out
}
