package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// insecureJWTSecret is the dev-only fallback. Production MUST override it.
const insecureJWTSecret = "dev-only-change-me-32-chars-min-len"

// Config is the root config object. Tagged for viper unmarshaling.
type Config struct {
	Env    string `mapstructure:"env"`
	Server struct {
		Addr           string   `mapstructure:"addr"`
		AllowedOrigins []string `mapstructure:"allowed_origins"`
	} `mapstructure:"server"`
	DB struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"db"`
	Redis struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"redis"`
	MinIO struct {
		Endpoint  string `mapstructure:"endpoint"`
		AccessKey string `mapstructure:"access_key"`
		SecretKey string `mapstructure:"secret_key"`
	} `mapstructure:"minio"`
	Logger struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logger"`
	Auth struct {
		JWTSecret string `mapstructure:"jwt_secret"`
	} `mapstructure:"auth"`
}

// Load reads configs/<env>.yaml + env overrides. Env vars: IOP_<SECTION>_<KEY>.
// Example: IOP_DB_DSN overrides db.dsn.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("dev")
	v.AddConfigPath("./configs")
	v.AddConfigPath("./server/configs")

	v.SetEnvPrefix("IOP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("env", "dev")
	v.SetDefault("server.addr", ":8080")
	v.SetDefault("server.allowed_origins", []string{"*"})
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "console")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: %w", err)
		}
		// Missing file is fine if env vars cover required fields.
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("config unmarshal: %w", err)
	}
	return &c, nil
}

// IsProd reports whether the config targets a non-development environment.
func (c *Config) IsProd() bool {
	env := strings.ToLower(strings.TrimSpace(c.Env))
	return env != "" && env != "dev" && env != "development" && env != "test"
}

// Validate enforces production-safety invariants. In dev it is lenient and only
// returns warnings (as a slice) for things that would be fatal in prod; in
// non-dev environments those become a hard error so the server fails fast at boot
// instead of running with an insecure default.
//
// Returns (warnings, error). Callers should log warnings and abort on error.
func (c *Config) Validate() (warnings []string, err error) {
	prod := c.IsProd()

	// JWT secret must be strong and never the public dev fallback in prod.
	secret := strings.TrimSpace(c.Auth.JWTSecret)
	switch {
	case secret == "" || secret == insecureJWTSecret:
		if prod {
			return warnings, fmt.Errorf("config: auth.jwt_secret must be set (env IOP_AUTH_JWT_SECRET) and at least 32 chars in env %q", c.Env)
		}
		warnings = append(warnings, "auth.jwt_secret is unset/insecure — using the dev fallback (NEVER do this in production)")
	case len(secret) < 32:
		if prod {
			return warnings, fmt.Errorf("config: auth.jwt_secret must be at least 32 chars in env %q (got %d)", c.Env, len(secret))
		}
		warnings = append(warnings, fmt.Sprintf("auth.jwt_secret is short (%d chars) — use >=32 in production", len(secret)))
	}

	// CORS must not be a wildcard in prod (credentials + cookies require a real allowlist).
	for _, o := range c.Server.AllowedOrigins {
		if strings.TrimSpace(o) == "*" {
			if prod {
				return warnings, fmt.Errorf("config: server.allowed_origins must not contain \"*\" in env %q — set explicit origins", c.Env)
			}
			warnings = append(warnings, "server.allowed_origins contains \"*\" — tighten to explicit origins in production")
		}
	}

	// Postgres should use TLS in prod.
	if strings.Contains(c.DB.DSN, "sslmode=disable") {
		if prod {
			return warnings, fmt.Errorf("config: db.dsn uses sslmode=disable in env %q — require TLS in production", c.Env)
		}
		warnings = append(warnings, "db.dsn uses sslmode=disable — use sslmode=require in production")
	}

	return warnings, nil
}

// ResolvedJWTSecret returns the secret to sign tokens with, applying the dev
// fallback only when not in production. Build() should use this instead of an
// inline fallback.
func (c *Config) ResolvedJWTSecret() string {
	s := strings.TrimSpace(c.Auth.JWTSecret)
	if s == "" && !c.IsProd() {
		return insecureJWTSecret
	}
	return s
}
