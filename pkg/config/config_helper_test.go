package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type testEnvConfig struct {
	Name    string  `mapstructure:"GOSDKCFG_NAME"    json:"gosdkcfg_name"`
	DB      int     `mapstructure:"GOSDKCFG_DB"      json:"gosdkcfg_db"`
	Small   int8    `mapstructure:"GOSDKCFG_SMALL"   json:"gosdkcfg_small"`
	Size    uint16  `mapstructure:"GOSDKCFG_SIZE"    json:"gosdkcfg_size"`
	Ratio   float64 `mapstructure:"GOSDKCFG_RATIO"   json:"gosdkcfg_ratio"`
	Enabled bool    `mapstructure:"GOSDKCFG_ENABLED" json:"gosdkcfg_enabled"`
	Limit   *int    `mapstructure:"GOSDKCFG_LIMIT"   json:"gosdkcfg_limit"`
	Port    port    `mapstructure:"GOSDKCFG_PORT"    json:"gosdkcfg_port"`
	Quoted  int     `mapstructure:"GOSDKCFG_QUOTED"  json:"gosdkcfg_quoted,string"`
	Level   level   `mapstructure:"GOSDKCFG_LEVEL"   json:"gosdkcfg_level"`
	NoJSON  int     `mapstructure:"GOSDKCFG_NO_JSON"`
	Skipped int     `mapstructure:"GOSDKCFG_SKIPPED" json:"-"`
}

type testDurationConfig struct {
	Timeout time.Duration `mapstructure:"GOSDKCFG_TIMEOUT" json:"gosdkcfg_timeout"`
}

type port int

// level Тип с UnmarshalText: строка из окружения передаётся ему как есть.
type level int

func (l *level) UnmarshalText(text []byte) error {
	if string(text) != "debug" {
		return errors.New("unknown level")
	}

	*l = 7

	return nil
}

func TestInitConfig_OSEnvTypedByField(t *testing.T) {
	loadEnvFile(t, "")
	t.Setenv("GOSDKCFG_NAME", "15")
	t.Setenv("GOSDKCFG_DB", "15")
	t.Setenv("GOSDKCFG_SMALL", "-8")
	t.Setenv("GOSDKCFG_SIZE", "0x10")
	t.Setenv("GOSDKCFG_RATIO", "0.25")
	t.Setenv("GOSDKCFG_ENABLED", "true")
	t.Setenv("GOSDKCFG_LIMIT", "3")
	t.Setenv("GOSDKCFG_PORT", "8080")

	cfg := &testEnvConfig{}
	if err := InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if cfg.Name != "15" {
		t.Errorf("Name = %q, want %q", cfg.Name, "15")
	}
	if cfg.DB != 15 {
		t.Errorf("DB = %d, want 15", cfg.DB)
	}
	if cfg.Small != -8 {
		t.Errorf("Small = %d, want -8", cfg.Small)
	}
	if cfg.Size != 16 {
		t.Errorf("Size = %d, want 16", cfg.Size)
	}
	if cfg.Ratio != 0.25 {
		t.Errorf("Ratio = %v, want 0.25", cfg.Ratio)
	}
	if !cfg.Enabled {
		t.Error("Enabled = false, want true")
	}
	if cfg.Limit == nil || *cfg.Limit != 3 {
		t.Errorf("Limit = %v, want 3", cfg.Limit)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
}

func TestInitConfig_StringFromOSEnv(t *testing.T) {
	loadEnvFile(t, "")
	t.Setenv("GOSDKCFG_NAME", "  service ")

	cfg := &testEnvConfig{}
	if err := InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if cfg.Name != "  service " {
		t.Errorf("Name = %q, want %q", cfg.Name, "  service ")
	}
}

func TestInitConfig_EnvFileOnly(t *testing.T) {
	loadEnvFile(t, "GOSDKCFG_NAME=file\nGOSDKCFG_DB=2\nGOSDKCFG_ENABLED=true\n")

	cfg := &testEnvConfig{}
	if err := InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if cfg.Name != "file" || cfg.DB != 2 || !cfg.Enabled {
		t.Errorf("got Name=%q DB=%d Enabled=%v, want file 2 true", cfg.Name, cfg.DB, cfg.Enabled)
	}
}

func TestInitConfig_OSEnvOverridesEnvFile(t *testing.T) {
	loadEnvFile(t, "GOSDKCFG_NAME=file\nGOSDKCFG_DB=2\nGOSDKCFG_RATIO=0.5\nGOSDKCFG_ENABLED=true\n")
	t.Setenv("GOSDKCFG_NAME", "env")
	t.Setenv("GOSDKCFG_DB", "15")
	t.Setenv("GOSDKCFG_ENABLED", "false")

	cfg := &testEnvConfig{}
	if err := InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if cfg.Name != "env" {
		t.Errorf("Name = %q, want env", cfg.Name)
	}
	if cfg.DB != 15 {
		t.Errorf("DB = %d, want 15", cfg.DB)
	}
	if cfg.Ratio != 0.5 {
		t.Errorf("Ratio = %v, want 0.5 from file", cfg.Ratio)
	}
	if cfg.Enabled {
		t.Error("Enabled = true, want false from env")
	}
}

func TestInitConfig_EmptyOSEnvIsZero(t *testing.T) {
	loadEnvFile(t, "GOSDKCFG_DB=2\nGOSDKCFG_ENABLED=true\n")
	t.Setenv("GOSDKCFG_DB", "")
	t.Setenv("GOSDKCFG_ENABLED", "")

	cfg := &testEnvConfig{}
	if err := InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if cfg.DB != 0 || cfg.Enabled {
		t.Errorf("got DB=%d Enabled=%v, want 0 false", cfg.DB, cfg.Enabled)
	}
}

func TestInitConfig_InvalidValue(t *testing.T) {
	cases := []struct {
		name string
		key  string
		val  string
	}{
		{name: "int", key: "GOSDKCFG_DB", val: "abc"},
		{name: "int overflow", key: "GOSDKCFG_SMALL", val: "300"},
		{name: "uint negative", key: "GOSDKCFG_SIZE", val: "-1"},
		{name: "float", key: "GOSDKCFG_RATIO", val: "1,5"},
		{name: "float NaN", key: "GOSDKCFG_RATIO", val: "NaN"},
		{name: "float Inf", key: "GOSDKCFG_RATIO", val: "+Inf"},
		{name: "bool", key: "GOSDKCFG_ENABLED", val: "yes"},
		{name: "pointer", key: "GOSDKCFG_LIMIT", val: "x"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			loadEnvFile(t, "")
			t.Setenv(tc.key, tc.val)

			err := InitConfig(&testEnvConfig{})
			if err == nil {
				t.Fatal("expected error")
			}
			if !strings.Contains(err.Error(), tc.key) {
				t.Errorf("error %q does not name %s", err, tc.key)
			}
		})
	}
}

func TestInitConfig_InvalidValueKeepsOtherFields(t *testing.T) {
	loadEnvFile(t, "")
	t.Setenv("GOSDKCFG_DB", "abc")
	t.Setenv("GOSDKCFG_NAME", "env")
	t.Setenv("GOSDKCFG_ENABLED", "true")

	cfg := &testEnvConfig{}
	if err := InitConfig(cfg); err == nil {
		t.Fatal("expected error")
	}

	if cfg.Name != "env" || !cfg.Enabled || cfg.DB != 0 {
		t.Errorf("got Name=%q Enabled=%v DB=%d, want env true 0", cfg.Name, cfg.Enabled, cfg.DB)
	}
}

func TestInitConfig_FieldsLeftAsBefore(t *testing.T) {
	loadEnvFile(t, "")
	t.Setenv("GOSDKCFG_QUOTED", "15")
	t.Setenv("GOSDKCFG_LEVEL", "debug")
	t.Setenv("GOSDKCFG_NO_JSON", "abc")
	t.Setenv("GOSDKCFG_SKIPPED", "abc")

	cfg := &testEnvConfig{}
	if err := InitConfig(cfg); err != nil {
		t.Fatalf("init config: %v", err)
	}

	if cfg.Quoted != 15 {
		t.Errorf("Quoted = %d, want 15", cfg.Quoted)
	}
	if cfg.Level != 7 {
		t.Errorf("Level = %d, want 7", cfg.Level)
	}
	if cfg.NoJSON != 0 || cfg.Skipped != 0 {
		t.Errorf("got NoJSON=%d Skipped=%d, want 0 0", cfg.NoJSON, cfg.Skipped)
	}
}

func TestInitConfig_DurationFromOSEnvAsBefore(t *testing.T) {
	loadEnvFile(t, "")
	t.Setenv("GOSDKCFG_TIMEOUT", "5s")

	err := InitConfig(&testDurationConfig{})

	if _, ok := err.(*json.UnmarshalTypeError); !ok {
		t.Fatalf("expected json type error as before, got %v", err)
	}
}

// loadEnvFile Загружает .env с содержимым content; пустой content — файла нет, только окружение.
func loadEnvFile(t *testing.T, content string) {
	t.Helper()

	path := filepath.Join(t.TempDir(), ".env")
	if content != "" {
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatalf("write env file: %v", err)
		}
	}

	if _, err := LoadEnv(path); err != nil {
		t.Fatalf("load env: %v", err)
	}
}
