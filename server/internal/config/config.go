package config

import (
	"fmt"
	"os"
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
		// AllowInsecure explicitly permits sslmode=disable in prod, for a DB on a
		// trusted private network (e.g. the bundled compose db). Default false.
		AllowInsecure bool `mapstructure:"allow_insecure"`
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

	// viper does not reliably coerce a string env var into a []string / bool via
	// Unmarshal, so apply these two env overrides explicitly. This lets the prod
	// compose set an explicit CORS allowlist (required, since "*" is rejected in
	// prod) and opt into an insecure DB link on a trusted private network.
	if raw := strings.TrimSpace(os.Getenv("IOP_SERVER_ALLOWED_ORIGINS")); raw != "" {
		var origins []string
		for _, o := range strings.Split(raw, ",") {
			if o = strings.TrimSpace(o); o != "" {
				origins = append(origins, o)
			}
		}
		c.Server.AllowedOrigins = origins
	}
	if raw := strings.TrimSpace(os.Getenv("IOP_DB_ALLOW_INSECURE")); raw != "" {
		c.DB.AllowInsecure = raw == "1" || strings.EqualFold(raw, "true") || strings.EqualFold(raw, "yes")
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

	// Postgres should use TLS in prod, unless the operator explicitly opts out for
	// a DB reached over a trusted private network (db.allow_insecure /
	// IOP_DB_ALLOW_INSECURE) — e.g. the bundled compose db on an internal-only network.
	if strings.Contains(c.DB.DSN, "sslmode=disable") {
		switch {
		case prod && !c.DB.AllowInsecure:
			return warnings, fmt.Errorf("config: db.dsn uses sslmode=disable in env %q — use sslmode=require in production, or set db.allow_insecure=true / IOP_DB_ALLOW_INSECURE=true if the DB is on a trusted private network", c.Env)
		case prod:
			warnings = append(warnings, "db.dsn uses sslmode=disable in prod — permitted only because db.allow_insecure is set; ensure the DB link is on a trusted private network")
		default:
			warnings = append(warnings, "db.dsn uses sslmode=disable — use sslmode=require in production")
		}
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
