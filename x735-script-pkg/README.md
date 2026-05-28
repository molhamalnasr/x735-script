# x735-script - Go Daemon for X735 Board

This package contains the compiled Go daemon (`x735-daemon`) that controls the fan speed and monitors the physical power button on the Geekworm X735 Power Management & Cooling Expansion Board.

## Configuration Layout

The configuration architecture is split into default settings and user overrides:

1.  **System Defaults (`/usr/share/x735/x735.default.conf`)**:
    *   Contains the default out-of-the-box parameters (PWM channel, Hz, sleep interval, default fan curves).
    *   **Do not modify this file.** It is overwritten on package upgrades.
2.  **User Overrides (`/etc/x735/x735.conf`)**:
    *   Contains your custom parameter overrides. 
    *   You only need to define the settings you want to change. All other keys fall back to the default file values.
3.  **Example Template (`/etc/x735/x735.conf.example`)**:
    *   A fully-commented example showing all available config parameters.

### Hardcoded Hardware Pins
To protect your Raspberry Pi and the X735 board from accidental hardware damage, the physical GPIO pin numbers are **hardcoded in the compiled binary** and cannot be configured or overridden via config files:
*   `SHUTDOWN_PIN` = 5 (GPIO 5 / Physical pin 29)
*   `BOOT_PIN` = 12 (GPIO 12 / Physical pin 32)
*   `BUTTON_PIN` = 20 (GPIO 20 / Physical pin 38)
*   `GPIO_CHIP` = 0

---

## Parameter Validation
The config loader validates each user override in `/etc/x735/x735.conf` on startup:
*   `PWM_CHANNEL` must be `0` or `1`.
*   `PWM_HERTZ` and `SLEEP_INTERVAL` must be greater than `0`.
*   `METRICS_PORT` must be between `1` and `65535`.
*   `REBOOT_PULSE_MAX` must be greater than `REBOOT_PULSE_MIN`.
*   `FAN_THRESHOLDS` and `FAN_DUTY_CYCLES` must have matching lengths, contain valid values, and thresholds must be in ascending order.

If any parameter is invalid, a warning is printed to standard error, and the daemon falls back to the safe default value from `/usr/share/x735/x735.default.conf` for that specific setting.

---

## Prometheus Metrics
If you set `ENABLE_METRICS=true` in `/etc/x735/x735.conf`, the daemon starts an HTTP metrics server on the port specified by `METRICS_PORT` (defaulting to `:9735`).

You can scrape this endpoint at:
`http://<your-pi-ip>:<metrics-port>/metrics`

### Exposed Metrics
*   `x735_cpu_temperature_celsius` (Gauge): Current CPU temperature.
*   `x735_fan_duty_cycle_percent` (Gauge): Current fan speed duty cycle.

---

## Services & Executables

*   `x735-fan.service`: Manages the CPU temperature polling and fan speeds. Calls `/usr/bin/x735-daemon fan`.
*   `x735-pwr.service`: Manages button click monitoring for safe shutdown/reboot. Calls `/usr/bin/x735-daemon pwr`.
*   `/usr/bin/x735off [seconds]`: Symbolic link pointing to `/usr/bin/x735-daemon`. Sends a 4-second pulse (or custom duration) to the X735 board to cut the main power after the OS has halted.

---

## Support & Wiki

*   User Guide Wiki: https://wiki.geekworm.com/X735-script
*   Email: support@geekworm.com
