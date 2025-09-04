package config

import (
	"github.com/spf13/pflag"
	"github.com/stretchr/testify/suite"
	"io"
	"os"
	"testing"
)

type ConfigSuite struct {
	suite.Suite
	prevEnv map[string]*string
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

func (s *ConfigSuite) SetupTest() {
	s.prevEnv = make(map[string]*string)
	for _, key := range []string{"RUN_ADDRESS", "DATABASE_URI", "ACCRUAL_SYSTEM_ADDRESS"} {
		if val, ok := os.LookupEnv(key); ok {
			tmp := val
			s.prevEnv[key] = &tmp
		} else {
			s.prevEnv[key] = nil
		}
		_ = os.Unsetenv(key)
	}

	resetFlags()
	os.Args = []string{"test"}
}

func (s *ConfigSuite) TearDownTest() {
	for key, val := range s.prevEnv {
		if val == nil {
			_ = os.Unsetenv(key)
		} else {
			_ = os.Setenv(key, *val)
		}
	}

	resetFlags()
	os.Args = []string{"test"}
}

func resetFlags() {
	pflag.CommandLine = pflag.NewFlagSet(os.Args[0], pflag.ContinueOnError)
	pflag.CommandLine.SetOutput(io.Discard)
}

func (s *ConfigSuite) TestDefaults_WhenOnlyRequiredEnvProvided() {
	s.Require().NoError(os.Setenv("DATABASE_URI", "postgres://user:pass@localhost:5432/db"))
	s.Require().NoError(os.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://accrual:8080"))

	cfg, err := LoadConfig(nil)
	s.Require().NoError(err)
	s.Equal("0.0.0.0:8080", cfg.ServerAddress)
	s.Equal("postgres://user:pass@localhost:5432/db", cfg.DatabaseUri)
	s.Equal("http://accrual:8080", cfg.AccuralAddress)
}

func (s *ConfigSuite) TestEnvOnly() {
	s.Require().NoError(os.Setenv("RUN_ADDRESS", "127.0.0.1:9000"))
	s.Require().NoError(os.Setenv("DATABASE_URI", "postgres://env/db"))
	s.Require().NoError(os.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://env-accrual"))

	cfg, err := LoadConfig(nil)
	s.Require().NoError(err)
	s.Equal("127.0.0.1:9000", cfg.ServerAddress)
	s.Equal("postgres://env/db", cfg.DatabaseUri)
	s.Equal("http://env-accrual", cfg.AccuralAddress)
}

func (s *ConfigSuite) TestFlagsOverrideEnv() {
	s.Require().NoError(os.Setenv("RUN_ADDRESS", "127.0.0.1:9000"))
	s.Require().NoError(os.Setenv("DATABASE_URI", "postgres://env/db"))
	s.Require().NoError(os.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://env-accrual"))

	resetFlags()
	os.Args = []string{
		"test",
		"-a", "0.0.0.0:7000",
		"-d", "postgres://flag/db",
		"-r", "http://flag-accrual",
	}

	cfg, err := LoadConfig(nil)
	s.Require().NoError(err)
	s.Equal("0.0.0.0:7000", cfg.ServerAddress)
	s.Equal("postgres://flag/db", cfg.DatabaseUri)
	s.Equal("http://flag-accrual", cfg.AccuralAddress)
}

func (s *ConfigSuite) TestLocalOverridesHaveMaxPriority() {
	s.Require().NoError(os.Setenv("RUN_ADDRESS", "127.0.0.1:9000"))
	s.Require().NoError(os.Setenv("DATABASE_URI", "postgres://env/db"))
	s.Require().NoError(os.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://env-accrual"))

	resetFlags()
	os.Args = []string{
		"test",
		"-a", "0.0.0.0:7000",
		"-d", "postgres://flag/db",
		"-r", "http://flag-accrual",
	}

	args := map[string]any{
		"ServerAddress":  "10.0.0.1:8081",
		"DatabaseUri":    "postgres://local/db",
		"AccuralAddress": "http://local-accrual",
	}

	cfg, err := LoadConfig(&args)
	s.Require().NoError(err)
	s.Equal("10.0.0.1:8081", cfg.ServerAddress)
	s.Equal("postgres://local/db", cfg.DatabaseUri)
	s.Equal("http://local-accrual", cfg.AccuralAddress)
}

func (s *ConfigSuite) TestPartialLocalOverrides() {
	s.Require().NoError(os.Setenv("RUN_ADDRESS", "127.0.0.1:9000"))
	s.Require().NoError(os.Setenv("DATABASE_URI", "postgres://env/db"))
	s.Require().NoError(os.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://env-accrual"))

	args := map[string]any{
		"ServerAddress": "10.10.10.10:9090",
	}

	cfg, err := LoadConfig(&args)
	s.Require().NoError(err)
	s.Equal("10.10.10.10:9090", cfg.ServerAddress)
	s.Equal("postgres://env/db", cfg.DatabaseUri)
	s.Equal("http://env-accrual", cfg.AccuralAddress)
}

func (s *ConfigSuite) TestErrorWhenDatabaseUriMissing() {
	s.Require().NoError(os.Setenv("ACCRUAL_SYSTEM_ADDRESS", "http://acc"))

	cfg, err := LoadConfig(nil)
	s.Require().Error(err)
	s.Nil(cfg)
	s.Contains(err.Error(), "DATABASE_URI is not set")
}

func (s *ConfigSuite) TestErrorWhenAccrualAddressMissing() {
	s.Require().NoError(os.Setenv("DATABASE_URI", "postgres://db"))

	cfg, err := LoadConfig(nil)
	s.Require().Error(err)
	s.Nil(cfg)
	s.Contains(err.Error(), "ACCRUAL_SYSTEM_ADDRESS is not set")
}
