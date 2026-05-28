package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Hardware fixed pins (not configurable by users)
const (
	FixedShutdownPin = 5
	FixedBootPin     = 12
	FixedButtonPin   = 20
	FixedGpioChip    = 0
)

// Config represents all application configurations.
type Config struct {
	PwmChannel     int
	PwmHertz       int
	SleepInterval  int
	ShowDebug      bool
	EnableMetrics  bool
	RebootPulseMin int // in milliseconds
	RebootPulseMax int // in milliseconds
	FanThresholds  []int
	FanDutyCycles  []int

	// Hardcoded pins (read-only)
	ShutdownPin int
	BootPin     int
	ButtonPin   int
	GpioChip    int
}

// NewDefaultConfig returns a configuration pre-filled with hardware-fixed pin constants and defaults.
func NewDefaultConfig() *Config {
	return &Config{
		PwmChannel:     1,
		PwmHertz:       2000,
		SleepInterval:  5,
		ShowDebug:      false,
		EnableMetrics:  false,
		RebootPulseMin: 200,
		RebootPulseMax: 600,
		FanThresholds:  []int{25, 40, 50, 60, 70, 75},
		FanDutyCycles:  []int{40, 45, 50, 70, 80, 100},

		// Fixed Pins
		ShutdownPin: FixedShutdownPin,
		BootPin:     FixedBootPin,
		ButtonPin:   FixedButtonPin,
		GpioChip:    FixedGpioChip,
	}
}

// LoadConfig loads the default config, reads user changes from /etc/x735/x735.conf,
// validates each parameter inline, and falls back to defaults for invalid/missing keys.
func LoadConfig(defaultPath, userPath string) *Config {
	// Initialize defaults
	cfg := NewDefaultConfig()

	// Load and validate from defaults file
	cfg = loadAndValidateConfig(defaultPath, cfg, false)

	// Force hardware pins to remain hardcoded
	cfg.ShutdownPin = FixedShutdownPin
	cfg.BootPin = FixedBootPin
	cfg.ButtonPin = FixedButtonPin
	cfg.GpioChip = FixedGpioChip

	// Load and validate from user file (if it exists)
	if _, err := os.Stat(userPath); err == nil {
		cfg = loadAndValidateConfig(userPath, cfg, true)
	}

	return cfg
}

func loadAndValidateConfig(path string, cfg *Config, isUser bool) *Config {
	file, err := os.Open(path)
	if err != nil {
		return cfg
	}
	defer file.Close()

	// Temporary holders for fan curve
	var tempThresholds []int
	var tempDutyCycles []int
	hasThresholds := false
	hasDutyCycles := false

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)

		switch key {
		case "PWM_CHANNEL":
			if parsed, err := strconv.Atoi(val); err == nil {
				if parsed == 0 || parsed == 1 {
					cfg.PwmChannel = parsed
				} else if isUser {
					fmt.Fprintf(os.Stderr, "Warning: Invalid PWM_CHANNEL (%d) in user config. Retaining default (%d).\n", parsed, cfg.PwmChannel)
				}
			}
		case "PWM_HERTZ":
			if parsed, err := strconv.Atoi(val); err == nil {
				if parsed > 0 {
					cfg.PwmHertz = parsed
				} else if isUser {
					fmt.Fprintf(os.Stderr, "Warning: Invalid PWM_HERTZ (%d) in user config. Retaining default (%d).\n", parsed, cfg.PwmHertz)
				}
			}
		case "SLEEP_INTERVAL":
			if parsed, err := strconv.Atoi(val); err == nil {
				if parsed > 0 {
					cfg.SleepInterval = parsed
				} else if isUser {
					fmt.Fprintf(os.Stderr, "Warning: Invalid SLEEP_INTERVAL (%d) in user config. Retaining default (%d).\n", parsed, cfg.SleepInterval)
				}
			}
		case "SHOW_DEBUG":
			cfg.ShowDebug = strings.ToLower(val) == "true" || val == "1"
		case "ENABLE_METRICS":
			cfg.EnableMetrics = strings.ToLower(val) == "true" || val == "1"
		case "REBOOT_PULSE_MIN":
			if parsed, err := strconv.Atoi(val); err == nil {
				if parsed > 0 {
					cfg.RebootPulseMin = parsed
				} else if isUser {
					fmt.Fprintf(os.Stderr, "Warning: Invalid REBOOT_PULSE_MIN (%d) in user config. Retaining default (%d).\n", parsed, cfg.RebootPulseMin)
				}
			}
		case "REBOOT_PULSE_MAX":
			if parsed, err := strconv.Atoi(val); err == nil {
				if parsed > cfg.RebootPulseMin {
					cfg.RebootPulseMax = parsed
				} else if isUser {
					fmt.Fprintf(os.Stderr, "Warning: REBOOT_PULSE_MAX (%d) must be greater than REBOOT_PULSE_MIN (%d). Retaining default (%d).\n", parsed, cfg.RebootPulseMin, cfg.RebootPulseMax)
				}
			}
		case "FAN_THRESHOLDS":
			tempThresholds = parseSlice(val)
			hasThresholds = true
		case "FAN_DUTY_CYCLES":
			tempDutyCycles = parseSlice(val)
			hasDutyCycles = true
		}
	}

	// Validate and apply fan curve if either was parsed
	if hasThresholds || hasDutyCycles {
		if !hasThresholds || !hasDutyCycles {
			if isUser {
				fmt.Fprintf(os.Stderr, "Warning: Both FAN_THRESHOLDS and FAN_DUTY_CYCLES must be defined to override default curve.\n")
			}
		} else {
			if len(tempThresholds) == len(tempDutyCycles) && len(tempThresholds) > 0 && isSorted(tempThresholds) && allBetween0And100(tempDutyCycles) {
				cfg.FanThresholds = tempThresholds
				cfg.FanDutyCycles = tempDutyCycles
			} else if isUser {
				fmt.Fprintf(os.Stderr, "Warning: Invalid user fan curve configuration. Retaining default fan curve.\n")
			}
		}
	}

	return cfg
}

func isSorted(slice []int) bool {
	for i := 1; i < len(slice); i++ {
		if slice[i] < slice[i-1] {
			return false
		}
	}
	return true
}

func allBetween0And100(slice []int) bool {
	for _, v := range slice {
		if v < 0 || v > 100 {
			return false
		}
	}
	return true
}

func parseSlice(val string) []int {
	fields := strings.Fields(strings.ReplaceAll(val, ",", " "))
	var res []int
	for _, f := range fields {
		if parsed, err := strconv.Atoi(f); err == nil {
			res = append(res, parsed)
		}
	}
	return res
}
