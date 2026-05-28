package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigLoadAndValidation(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "x735-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	defaultPath := filepath.Join(tempDir, "default.conf")
	userPath := filepath.Join(tempDir, "user.conf")

	// 1. Write mock default config
	defaultContent := `
PWM_CHANNEL=1
PWM_HERTZ=2000
SLEEP_INTERVAL=5
SHOW_DEBUG=false
ENABLE_METRICS=false
REBOOT_PULSE_MIN=200
REBOOT_PULSE_MAX=600
FAN_THRESHOLDS=25 40 50
FAN_DUTY_CYCLES=40 45 50
`
	if err := os.WriteFile(defaultPath, []byte(defaultContent), 0644); err != nil {
		t.Fatalf("failed to write default config: %v", err)
	}

	// 2. Test initial load (user config missing)
	cfg := LoadConfig(defaultPath, userPath)
	if cfg.PwmChannel != 1 || cfg.PwmHertz != 2000 || cfg.SleepInterval != 5 || cfg.EnableMetrics {
		t.Errorf("Expected defaults, got %+v", cfg)
	}

	// 3. Write user config with some overrides and one invalid override
	userContent := `
PWM_CHANNEL=0
PWM_HERTZ=-500 # Invalid, should revert to 2000
SLEEP_INTERVAL=10
ENABLE_METRICS=true
SHUTDOWN_PIN=99 # Pins not configurable, should be ignored
`
	if err := os.WriteFile(userPath, []byte(userContent), 0644); err != nil {
		t.Fatalf("failed to write user config: %v", err)
	}

	cfg = LoadConfig(defaultPath, userPath)

	// User overrides applied
	if cfg.PwmChannel != 0 {
		t.Errorf("Expected PWM_CHANNEL to be 0, got %d", cfg.PwmChannel)
	}
	if cfg.SleepInterval != 10 {
		t.Errorf("Expected SLEEP_INTERVAL to be 10, got %d", cfg.SleepInterval)
	}
	if !cfg.EnableMetrics {
		t.Errorf("Expected ENABLE_METRICS to be true, got %t", cfg.EnableMetrics)
	}

	// Invalid value reverted to default
	if cfg.PwmHertz != 2000 {
		t.Errorf("Expected invalid PWM_HERTZ to revert to default 2000, got %d", cfg.PwmHertz)
	}

	// Pins remain hardcoded
	if cfg.ShutdownPin != FixedShutdownPin {
		t.Errorf("Expected ShutdownPin to remain fixed %d, got %d", FixedShutdownPin, cfg.ShutdownPin)
	}
}
