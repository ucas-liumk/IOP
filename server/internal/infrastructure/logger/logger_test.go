package logger

import (
	"bytes"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestNew_WritesJSON(t *testing.T) {
	var buf bytes.Buffer
	l := newTestable(&buf, zapcore.DebugLevel)
	l.Info("hello")
	if !bytes.Contains(buf.Bytes(), []byte(`"msg":"hello"`)) {
		t.Fatalf("expected JSON output, got: %s", buf.String())
	}
}

func TestSanitize_RedactsPasswords(t *testing.T) {
	in := map[string]string{
		"username": "alice",
		"password": "supersecret",
		"token":    "abc.def.ghi",
	}
	out := Sanitize(in)
	m, _ := out.(map[string]string)
	if m["password"] != "[REDACTED]" {
		t.Fatalf("password not redacted: %v", m)
	}
	if m["token"] != "[REDACTED]" {
		t.Fatalf("token not redacted: %v", m)
	}
	if m["username"] != "alice" {
		t.Fatalf("username changed: %v", m)
	}
}
