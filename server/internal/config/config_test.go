package config

import "testing"

// mkConfig builds a Config with the given env + DSN + allow-insecure, leaving a
// strong JWT secret and an explicit CORS origin so only the property under test
// can trip Validate.
func mkConfig(env, dsn string, allowInsecure bool) *Config {
	c := &Config{Env: env}
	c.DB.DSN = dsn
	c.DB.AllowInsecure = allowInsecure
	c.Auth.JWTSecret = "this-is-a-sufficiently-long-jwt-secret-key"
	c.Server.AllowedOrigins = []string{"https://iop.example.com"}
	return c
}

func TestValidate_SSLMode(t *testing.T) {
	cases := []struct {
		name          string
		env           string
		dsn           string
		allowInsecure bool
		wantErr       bool
	}{
		{"dev tolerates sslmode=disable", "dev", "postgres://x/y?sslmode=disable", false, false},
		{"prod rejects sslmode=disable by default", "prod", "postgres://x/y?sslmode=disable", false, true},
		{"prod allows sslmode=disable with allow_insecure", "prod", "postgres://x/y?sslmode=disable", true, false},
		{"prod accepts sslmode=require", "prod", "postgres://x/y?sslmode=require", false, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mkConfig(tc.env, tc.dsn, tc.allowInsecure).Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tc.wantErr)
			}
		})
	}
}

func TestValidate_CORSWildcard(t *testing.T) {
	// Explicit origins pass in prod.
	c := mkConfig("prod", "postgres://x/y?sslmode=require", false)
	if _, err := c.Validate(); err != nil {
		t.Fatalf("explicit origins should pass in prod: %v", err)
	}
	// Wildcard is fatal in prod, a warning in dev.
	c.Server.AllowedOrigins = []string{"*"}
	if _, err := c.Validate(); err == nil {
		t.Fatal("wildcard origins must be rejected in prod")
	}
	c.Env = "dev"
	if warns, err := c.Validate(); err != nil || len(warns) == 0 {
		t.Fatalf("wildcard origins in dev should warn, not error: warns=%v err=%v", warns, err)
	}
}
