package pwm

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// PWMController defines the interface for interacting with hardware PWM.
type PWMController interface {
	Init() error
	SetDutyCycle(dutyPercent int) error
	Close() error
}

// SysfsPWMController implements PWMController using the Linux sysfs interface.
type SysfsPWMController struct {
	chipPath    string
	channel     int
	frequencyHz int
	periodNs    int64
	autoCleanup bool
}

// NewSysfsPWMController creates a new SysfsPWMController.
func NewSysfsPWMController(channel int, frequencyHz int, autoCleanup bool) *SysfsPWMController {
	return &SysfsPWMController{
		channel:     channel,
		frequencyHz: frequencyHz,
		autoCleanup: autoCleanup,
	}
}

// Init auto-detects the pwmchip, exports the channel, and sets the initial frequency.
func (s *SysfsPWMController) Init() error {
	// 1. Auto-detect active pwmchip path
	err := s.detectChipPath()
	if err != nil {
		return err
	}

	// 2. Export the channel if not already exported
	channelPath := filepath.Join(s.chipPath, fmt.Sprintf("pwm%d", s.channel))
	if _, err := os.Stat(channelPath); os.IsNotExist(err) {
		exportPath := filepath.Join(s.chipPath, "export")
		err = os.WriteFile(exportPath, []byte(strconv.Itoa(s.channel)), 0200)
		if err != nil {
			return fmt.Errorf("failed to export PWM channel %d: %w", s.channel, err)
		}
		// Give the kernel a small moment to create sysfs nodes
		time.Sleep(100 * time.Millisecond)
	}

	// Calculate period in nanoseconds
	s.periodNs = int64(1e9 / float64(s.frequencyHz))

	// 3. Set period (always set duty cycle to 0 first to avoid duty > period validation errors)
	err = s.writeChannelFile("duty_cycle", "0")
	if err != nil {
		return fmt.Errorf("failed to reset duty cycle: %w", err)
	}

	err = s.writeChannelFile("period", strconv.FormatInt(s.periodNs, 10))
	if err != nil {
		return fmt.Errorf("failed to set PWM period: %w", err)
	}

	// 4. Enable PWM
	err = s.writeChannelFile("enable", "1")
	if err != nil {
		return fmt.Errorf("failed to enable PWM: %w", err)
	}

	return nil
}

// SetDutyCycle sets the duty cycle as a percentage (0 to 100).
func (s *SysfsPWMController) SetDutyCycle(dutyPercent int) error {
	if dutyPercent < 0 || dutyPercent > 100 {
		return fmt.Errorf("duty cycle percentage must be between 0 and 100")
	}

	newDutyNs := s.periodNs * int64(dutyPercent) / 100
	err := s.writeChannelFile("duty_cycle", strconv.FormatInt(newDutyNs, 10))
	if err != nil {
		return fmt.Errorf("failed to set duty cycle to %d%% (%d ns): %w", dutyPercent, newDutyNs, err)
	}

	return nil
}

// Close disables and optionally unexports the PWM channel on shutdown.
func (s *SysfsPWMController) Close() error {
	// Set duty cycle to 0 first
	_ = s.SetDutyCycle(0)
	// Disable
	_ = s.writeChannelFile("enable", "0")

	if s.autoCleanup {
		unexportPath := filepath.Join(s.chipPath, "unexport")
		_ = os.WriteFile(unexportPath, []byte(strconv.Itoa(s.channel)), 0200)
	}
	return nil
}

// detectChipPath scans for pwmchip0, pwmchip1, etc.
func (s *SysfsPWMController) detectChipPath() error {
	matches, err := filepath.Glob("/sys/class/pwm/pwmchip*")
	if err != nil || len(matches) == 0 {
		return fmt.Errorf("no hardware PWM chips found in /sys/class/pwm/; make sure 'dtoverlay=pwm-2chan' is loaded")
	}
	// Select the first available pwmchip
	s.chipPath = matches[0]
	return nil
}

// writeChannelFile writes a string value to a file within the exported pwm channel directory.
func (s *SysfsPWMController) writeChannelFile(filename string, value string) error {
	filePath := filepath.Join(s.chipPath, fmt.Sprintf("pwm%d", s.channel), filename)
	return os.WriteFile(filePath, []byte(value), 0644)
}
