package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

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
