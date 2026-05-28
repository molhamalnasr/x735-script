package system

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// SystemOps defines system-level operations.
type SystemOps interface {
	ReadCPUTemperature() (float64, error)
	Reboot() error
	PowerOff() error
}

// RealSystemOps implements SystemOps using actual OS files and commands.
type RealSystemOps struct {
	tempPath string
}

// NewRealSystemOps creates a new RealSystemOps.
func NewRealSystemOps() *RealSystemOps {
	return &RealSystemOps{
		tempPath: "/sys/class/thermal/thermal_zone0/temp",
	}
}

// ReadCPUTemperature reads raw milli-degrees Celsius from thermal zone and returns float64.
func (s *RealSystemOps) ReadCPUTemperature() (float64, error) {
	data, err := os.ReadFile(s.tempPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read temperature file: %w", err)
	}

	trimmed := strings.TrimSpace(string(data))
	milliDeg, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("failed to parse temperature raw value '%s': %w", trimmed, err)
	}

	return float64(milliDeg) / 1000.0, nil
}

// Reboot runs the OS reboot command.
func (s *RealSystemOps) Reboot() error {
	cmd := exec.Command("reboot")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// PowerOff runs the OS poweroff command.
func (s *RealSystemOps) PowerOff() error {
	cmd := exec.Command("poweroff")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
