package config

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type AppConfig struct {
	ServerAddress  string `env:"RUN_ADDRESS"`
	DatabaseUri    string `env:"DATABASE_URI"`
	AccuralAddress string `env:"ACCRUAL_SYSTEM_ADDRESS"`
	MigrationDir   string `env:"MIGRATIONS_DIR"`
}

type ConfigError struct {
	Msg string
	err error
}

func Error(format string, a ...any) error {
	return &ConfigError{
		Msg: fmt.Sprintf(format, a...),
	}
}

func (e *ConfigError) Error() string {
	if e.err != nil && e.err.Error() != e.Error() {

		return fmt.Sprintf("[CONFIG] %s (previous: %v)", e.Msg, e.err)
	}

	return fmt.Sprintf("[CONFIG] %s", e.Msg)
}

func (e *ConfigError) Unwrap() error {
	return e.err
}

func LoadConfig(args *map[string]any) (*AppConfig, error) {
	v := viper.New()
	v.SetDefault("RUN_ADDRESS", "0.0.0.0:8080")
	v.SetDefault("MIGRATIONS_DIR", "./migrations")

	_ = v.BindEnv("RUN_ADDRESS")
	_ = v.BindEnv("DATABASE_URI")
	_ = v.BindEnv("ACCRUAL_SYSTEM_ADDRESS")
	_ = v.BindEnv("MIGRATIONS_DIR")
	v.AutomaticEnv()

	pflag.StringP("address", "a", "", "server address and port (e.g., 0.0.0.0:8080)")
	pflag.StringP("database", "d", "", "database connection URI")
	pflag.StringP("accrual", "r", "", "accrual system address")
	pflag.StringP("migrations", "m", "", "migrations directory path")

	if !pflag.Parsed() {
		pflag.Parse()
	}

	_ = v.BindPFlag("RUN_ADDRESS", pflag.Lookup("address"))
	_ = v.BindPFlag("DATABASE_URI", pflag.Lookup("database"))
	_ = v.BindPFlag("ACCRUAL_SYSTEM_ADDRESS", pflag.Lookup("accrual"))
	_ = v.BindPFlag("MIGRATIONS_DIR", pflag.Lookup("migrations"))

	cfg := &AppConfig{
		ServerAddress:  v.GetString("RUN_ADDRESS"),
		DatabaseUri:    v.GetString("DATABASE_URI"),
		AccuralAddress: v.GetString("ACCRUAL_SYSTEM_ADDRESS"),
		MigrationDir:   v.GetString("MIGRATIONS_DIR"),
	}

	if args != nil {
		redefineLocal(args, cfg)
	}

	if cfg.DatabaseUri == "" {
		return nil, Error("DATABASE_URI is not set")
	}

	if cfg.AccuralAddress == "" {
		return nil, Error("ACCRUAL_SYSTEM_ADDRESS is not set")
	}

	return cfg, nil
}

func redefineLocal(args *map[string]any, cfg *AppConfig) {
	if val, ok := (*args)["ServerAddress"]; ok {
		if strVal, ok := val.(string); ok {
			cfg.ServerAddress = strVal
		}
	}

	if val, ok := (*args)["DatabaseUri"]; ok {
		if strVal, ok := val.(string); ok {
			cfg.DatabaseUri = strVal
		}
	}

	if val, ok := (*args)["AccuralAddress"]; ok {
		if strVal, ok := val.(string); ok {
			cfg.AccuralAddress = strVal
		}
	}
	if val, ok := (*args)["MigrationDir"]; ok {
		if strVal, ok := val.(string); ok {
			cfg.MigrationDir = strVal
		}
	}
}
