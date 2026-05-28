package main

import (
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/molhamalnasr/x735-script/config"
	"github.com/molhamalnasr/x735-script/domain/fan"
	"github.com/molhamalnasr/x735-script/domain/power"
	"github.com/molhamalnasr/x735-script/infrastructure/gpio"
	"github.com/molhamalnasr/x735-script/infrastructure/pwm"
	"github.com/molhamalnasr/x735-script/infrastructure/system"
	"github.com/warthog618/go-gpiocdev"
)

const (
	defaultConfigPath = "/usr/share/x735/x735.default.conf"
	userConfigPath    = "/etc/x735/x735.conf"
)

// Global metrics variables
var (
	metricsMu     sync.RWMutex
	metricCpuTemp float64
	metricFanDuty int
)

func main() {
	// 1. Determine operation mode
	mode := getOperationMode()

	// 2. Resolve paths for development fallback
	defPath := defaultConfigPath
	usrPath := userConfigPath
	if _, err := os.Stat(defPath); os.IsNotExist(err) {
		defPath = "./x735-script-pkg/usr/share/x735/x735.default.conf"
	}
	if _, err := os.Stat(usrPath); os.IsNotExist(err) {
		usrPath = "./x735-script-pkg/etc/x735/x735.conf"
	}

	// 3. Load configuration
	cfg := config.LoadConfig(defPath, usrPath)

	// 4. Dispatch to specific sub-systems
	switch mode {
	case "fan":
		runFan(cfg)
	case "pwr":
		runPwr(cfg)
	case "off":
		runOff(cfg)
	default:
		printUsage()
		os.Exit(1)
	}
}

// getOperationMode checks binary symlink name and CLI arguments to determine target command.
func getOperationMode() string {
	execName := filepath.Base(os.Args[0])
	if execName == "x735off" {
		return "off"
	}

	if len(os.Args) >= 2 {
		arg := strings.ToLower(os.Args[1])
		if arg == "fan" || arg == "pwr" || arg == "off" {
			return arg
		}
	}
	return ""
}

func printUsage() {
	fmt.Printf("Usage: %s [fan|pwr|off]\n", filepath.Base(os.Args[0]))
	fmt.Println("  fan - Start the CPU temperature monitoring and fan speed daemon")
	fmt.Println("  pwr - Start the physical safe shutdown and reboot button monitor")
	fmt.Println("  off - Trigger a hardware safe power cut sequence (equivalent to x735off)")
}

func runFan(cfg *config.Config) {
	fmt.Println("Starting x735-daemon in fan control mode...")

	// Initialize Domain Controller (Hysteresis = 2.0°C)
	controller := fan.NewHysteresisController(cfg.FanThresholds, cfg.FanDutyCycles, 2.0)

	// Initialize Infrastructure Components
	pwmCtrl := pwm.NewSysfsPWMController(cfg.PwmChannel, cfg.PwmHertz, true)
	sysOps := system.NewRealSystemOps()

	err := pwmCtrl.Init()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing PWM hardware: %v\n", err)
		os.Exit(1)
	}
	defer pwmCtrl.Close()

	// Start Prometheus metrics server if enabled
	if cfg.EnableMetrics {
		startMetricsServer(cfg.MetricsPort)
	}

	// Handle signals for cleanup
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(cfg.SleepInterval) * time.Second)
	defer ticker.Stop()

	// Run initial temperature check
	updateFanSpeed(cfg, controller, pwmCtrl, sysOps)

	for {
		select {
		case <-ticker.C:
			updateFanSpeed(cfg, controller, pwmCtrl, sysOps)
		case sig := <-sigChan:
			fmt.Printf("Received signal %v. Cleaning up PWM and exiting...\n", sig)
			return
		}
	}
}

func updateFanSpeed(cfg *config.Config, controller *fan.HysteresisController, pwmCtrl pwm.PWMController, sysOps system.SystemOps) {
	temp, err := sysOps.ReadCPUTemperature()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading CPU temperature: %v\n", err)
		return
	}

	prevDuty := controller.GetCurrentDuty()
	targetDuty := controller.CalculateDutyCycle(temp)

	if targetDuty != prevDuty {
		err = pwmCtrl.SetDutyCycle(targetDuty)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error setting PWM duty cycle: %v\n", err)
			return
		}
		fmt.Printf("Fan speed changed to %d%%, CPU temp is %.2f°C\n", targetDuty, temp)
	} else if cfg.ShowDebug {
		fmt.Printf("[DEBUG] CPU temp: %.2f°C, Fan speed: %d%%\n", temp, targetDuty)
	}

	// Update metrics (even if values haven't changed)
	metricsMu.Lock()
	metricCpuTemp = temp
	metricFanDuty = targetDuty
	metricsMu.Unlock()
}

func runPwr(cfg *config.Config) {
	fmt.Println("Starting x735-daemon in power button monitoring mode...")

	// Initialize Domain Classifier
	classifier := power.NewPulseClassifier(cfg.RebootPulseMin, cfg.RebootPulseMax)

	// Initialize Infrastructure Components
	gpioMgr := gpio.NewGPIOManager(cfg.GpioChip)
	defer gpioMgr.Close()
	sysOps := system.NewRealSystemOps()

	// 1. Initialize boot pin to high (tells X735 board that RPi is booted)
	err := gpioMgr.RequestOutput(cfg.BootPin, 1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error configuring BOOT pin (GPIO %d): %v\n", cfg.BootPin, err)
		os.Exit(1)
	}

	// 2. Set up event handler for shutdown button clicks (GPIO 5)
	var (
		pressStart time.Time
		isPressed  bool
		mu         sync.Mutex
	)

	handler := func(ev gpiocdev.LineEvent) {
		mu.Lock()
		defer mu.Unlock()

		if ev.Type == gpiocdev.LineEventRisingEdge {
			if !isPressed {
				isPressed = true
				pressStart = time.Now()
				if cfg.ShowDebug {
					fmt.Printf("[DEBUG] Button pressed at %v\n", pressStart)
				}

				// Spawn a background timer. If the button is held down longer than
				// RebootPulseMax, trigger poweroff immediately without waiting for release.
				go func(start time.Time) {
					time.Sleep(time.Duration(cfg.RebootPulseMax) * time.Millisecond)
					mu.Lock()
					defer mu.Unlock()

					if isPressed && pressStart.Equal(start) {
						// Double-check if the pin is still active
						val, err := gpioMgr.GetValue(cfg.ShutdownPin)
						if err == nil && val == 1 {
							fmt.Printf("Your device is shutting down (GPIO %d), halting RPi...\n", cfg.ShutdownPin)
							isPressed = false
							_ = sysOps.PowerOff()
						}
					}
				}(pressStart)
			}
		} else if ev.Type == gpiocdev.LineEventFallingEdge {
			if isPressed {
				elapsed := time.Since(pressStart)
				isPressed = false
				if cfg.ShowDebug {
					fmt.Printf("[DEBUG] Button released after %v\n", elapsed)
				}

				action := classifier.Classify(elapsed)
				switch action {
				case power.ActionReboot:
					fmt.Printf("Your device is rebooting (GPIO %d), recycling RPi...\n", cfg.ShutdownPin)
					_ = sysOps.Reboot()
				case power.ActionPowerOff:
					fmt.Printf("Your device is shutting down (GPIO %d), halting RPi...\n", cfg.ShutdownPin)
					_ = sysOps.PowerOff()
				}
			}
		}
	}

	// 3. Register the event handler for edge changes
	err = gpioMgr.WatchEdge(cfg.ShutdownPin, handler)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error watching shutdown pin (GPIO %d): %v\n", cfg.ShutdownPin, err)
		os.Exit(1)
	}

	fmt.Println("Listening to shutdown button clicks...")

	// Listen for termination signals to cleanup GPIO state
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	<-sigChan
	fmt.Println("Shutting down power monitor...")
}

func runOff(cfg *config.Config) {
	// Parse sleep duration from command line args if provided
	sleepDuration := 4 * time.Second
	args := os.Args

	var sleepStr string
	if len(args) >= 2 && args[1] != "off" && args[1] != "fan" && args[1] != "pwr" {
		sleepStr = args[1]
	} else if len(args) >= 3 && args[1] == "off" {
		sleepStr = args[2]
	}

	if sleepStr != "" {
		if seconds, err := strconv.ParseFloat(sleepStr, 64); err == nil {
			sleepDuration = time.Duration(seconds * float64(time.Second))
		}
	}

	// Broadcast shutdown message to active terminal sessions
	wallMsg := fmt.Sprintf("Your device will shut down in %.0f seconds...", sleepDuration.Seconds())
	fmt.Println(wallMsg)
	cmd := exec.Command("wall")
	cmd.Stdin = strings.NewReader(wallMsg)
	_ = cmd.Run()

	// Initialize GPIO manager and set GPIO 20 high to trigger board safe shutdown
	gpioMgr := gpio.NewGPIOManager(cfg.GpioChip)
	defer gpioMgr.Close()

	err := gpioMgr.RequestOutput(cfg.ButtonPin, 1)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error configuring BUTTON pin (GPIO %d): %v\n", cfg.ButtonPin, err)
		os.Exit(1)
	}

	time.Sleep(sleepDuration)

	// Clean up: turn off BUTTON pulse
	_ = gpioMgr.SetValue(cfg.ButtonPin, 0)
}

func startMetricsServer(port int) {
	http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		metricsMu.RLock()
		defer metricsMu.RUnlock()

		w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
		fmt.Fprintln(w, "# HELP x735_cpu_temperature_celsius Current CPU temperature in Celsius")
		fmt.Fprintln(w, "# TYPE x735_cpu_temperature_celsius gauge")
		fmt.Fprintf(w, "x735_cpu_temperature_celsius %.2f\n", metricCpuTemp)

		fmt.Fprintln(w, "\n# HELP x735_fan_duty_cycle_percent Current fan speed duty cycle percent")
		fmt.Fprintln(w, "# TYPE x735_fan_duty_cycle_percent gauge")
		fmt.Fprintf(w, "x735_fan_duty_cycle_percent %d\n", metricFanDuty)
	})

	go func() {
		fmt.Printf("Starting Prometheus metrics server on :%d/metrics...\n", port)
		if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error starting metrics server: %v\n", err)
		}
	}()
}
