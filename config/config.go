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
// validates each parameter, and falls back to defaults for invalid keys.
func LoadConfig(defaultPath, userPath string) *Config {
	// 1. Load the defaults from the new default config file
	defaultCfg := loadRawConfig(defaultPath, NewDefaultConfig())

	// Force hardware pins to remain hardcoded
	defaultCfg.ShutdownPin = FixedShutdownPin
	defaultCfg.BootPin = FixedBootPin
	defaultCfg.ButtonPin = FixedButtonPin
	defaultCfg.GpioChip = FixedGpioChip

	// If no user config file exists, return the default config directly
	if _, err := os.Stat(userPath); os.IsNotExist(err) {
		return defaultCfg
	}

	// 2. Load user config raw entries
	userCfg := loadRawConfig(userPath, nil)
	if userCfg == nil {
		return defaultCfg
	}

	// 3. Validate user values and merge them into defaultCfg
	mergeAndValidate(defaultCfg, userCfg)

	return defaultCfg
}

// loadRawConfig parses a configuration file. If startWith is provided, it uses it as a template.
func loadRawConfig(path string, template *Config) *Config {
	var cfg *Config
	if template != nil {
		// Clone template values
		clone := *template
		cfg = &clone
	} else {
		cfg = &Config{}
	}

	file, err := os.Open(path)
	if err != nil {
		return cfg
	}
	defer file.Close()

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
				cfg.PwmChannel = parsed
			}
		case "PWM_HERTZ":
			if parsed, err := strconv.Atoi(val); err == nil {
				cfg.PwmHertz = parsed
			}
		case "SLEEP_INTERVAL":
			if parsed, err := strconv.Atoi(val); err == nil {
				cfg.SleepInterval = parsed
			}
		case "SHOW_DEBUG":
			cfg.ShowDebug = strings.ToLower(val) == "true" || val == "1"
		case "ENABLE_METRICS":
			cfg.EnableMetrics = strings.ToLower(val) == "true" || val == "1"
		case "REBOOT_PULSE_MIN":
			if parsed, err := strconv.Atoi(val); err == nil {
				cfg.RebootPulseMin = parsed
			}
		case "REBOOT_PULSE_MAX":
			if parsed, err := strconv.Atoi(val); err == nil {
				cfg.RebootPulseMax = parsed
			}
		case "FAN_THRESHOLDS":
			cfg.FanThresholds = parseSlice(val)
		case "FAN_DUTY_CYCLES":
			cfg.FanDutyCycles = parseSlice(val)
		}
	}
	return cfg
}

// mergeAndValidate merges user config into defaults, resetting invalid fields to defaults.
func mergeAndValidate(def *Config, user *Config) {
	// Validate PWM Channel
	if user.PwmChannel == 0 || user.PwmChannel == 1 {
		def.PwmChannel = user.PwmChannel
	} else if user.PwmChannel != 0 { // user.PwmChannel is initialized (not zero-value fallback)
		fmt.Fprintf(os.Stderr, "Warning: Invalid PWM_CHANNEL (%d) in user config. Falling back to default (%d).\n", user.PwmChannel, def.PwmChannel)
	}

	// Validate PWM Hertz
	if user.PwmHertz > 0 {
		def.PwmHertz = user.PwmHertz
	} else if user.PwmHertz != 0 {
		fmt.Fprintf(os.Stderr, "Warning: Invalid PWM_HERTZ (%d). Falling back to default (%d).\n", user.PwmHertz, def.PwmHertz)
	}

	// Validate Sleep Interval
	if user.SleepInterval > 0 {
		def.SleepInterval = user.SleepInterval
	} else if user.SleepInterval != 0 {
		fmt.Fprintf(os.Stderr, "Warning: Invalid SLEEP_INTERVAL (%d). Falling back to default (%d).\n", user.SleepInterval, def.SleepInterval)
	}

	// Validate Reboot Pulse Min
	if user.RebootPulseMin > 0 {
		def.RebootPulseMin = user.RebootPulseMin
	} else if user.RebootPulseMin != 0 {
		fmt.Fprintf(os.Stderr, "Warning: Invalid REBOOT_PULSE_MIN (%d). Falling back to default (%d).\n", user.RebootPulseMin, def.RebootPulseMin)
	}

	// Validate Reboot Pulse Max
	if user.RebootPulseMax > 0 {
		// Reboot pulse max must be greater than reboot pulse min
		if user.RebootPulseMax > def.RebootPulseMin {
			def.RebootPulseMax = user.RebootPulseMax
		} else {
			fmt.Fprintf(os.Stderr, "Warning: REBOOT_PULSE_MAX (%d) must be greater than REBOOT_PULSE_MIN (%d). Falling back to default (%d).\n", user.RebootPulseMax, def.RebootPulseMin, def.RebootPulseMax)
		}
	}

	// Booleans (always valid)
	def.ShowDebug = user.ShowDebug
	def.EnableMetrics = user.EnableMetrics

	// Validate Fan curve
	if len(user.FanThresholds) > 0 || len(user.FanDutyCycles) > 0 {
		if len(user.FanThresholds) == len(user.FanDutyCycles) && len(user.FanThresholds) > 0 && isSorted(user.FanThresholds) && allBetween0And100(user.FanDutyCycles) {
			def.FanThresholds = user.FanThresholds
			def.FanDutyCycles = user.FanDutyCycles
		} else {
			fmt.Fprintf(os.Stderr, "Warning: Invalid user fan curve configuration. Falling back to default thresholds %v and duty cycles %v.\n", def.FanThresholds, def.FanDutyCycles)
		}
	}
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
