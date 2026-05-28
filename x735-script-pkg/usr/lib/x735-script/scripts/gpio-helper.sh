#!/usr/bin/env bash

# Auto-detect GPIO access method: gpiod_v2 (Debian 13), gpiod_v1 (Debian 12), or sysfs (legacy fallback)
detect_gpio_mode() {
    if which gpioget >/dev/null 2>&1 && which gpioset >/dev/null 2>&1; then
        local version
        version=$(gpioget -v 2>&1 | grep -oE '[0-9]+\.[0-9]+' | head -n1)
        if [[ "$version" =~ ^2\. ]]; then
            echo "gpiod_v2"
        else
            echo "gpiod_v1"
        fi
    elif [ -d "/sys/class/gpio" ]; then
        echo "sysfs"
    else
        echo "unknown"
    fi
}

GPIO_MODE=$(detect_gpio_mode)
# Store background gpioset PIDs to kill them on cleanup (primarily for gpiod_v2)
declare -A GPIOD_PIDS

gpio_init_input() {
    local chip="$1"
    local pin="$2"

    if [ "$GPIO_MODE" = "sysfs" ]; then
        if [ ! -d "/sys/class/gpio/gpio${pin}" ]; then
            echo "$pin" > /sys/class/gpio/export
        fi
        echo "in" > "/sys/class/gpio/gpio${pin}/direction"
    elif [ "$GPIO_MODE" = "gpiod_v1" ]; then
        # In libgpiod v1, no explicit initialization is needed for input
        true
    elif [ "$GPIO_MODE" = "gpiod_v2" ]; then
        # In libgpiod v2, no explicit initialization is needed for input
        true
    else
        echo "Error: Unknown GPIO mode or tools not installed." >&2
        return 1
    fi
}

gpio_init_output() {
    local chip="$1"
    local pin="$2"
    local val="$3"

    if [ "$GPIO_MODE" = "sysfs" ]; then
        if [ ! -d "/sys/class/gpio/gpio${pin}" ]; then
            echo "$pin" > /sys/class/gpio/export
        fi
        echo "out" > "/sys/class/gpio/gpio${pin}/direction"
        echo "$val" > "/sys/class/gpio/gpio${pin}/value"
    elif [ "$GPIO_MODE" = "gpiod_v1" ]; then
        gpioset "$chip" "$pin=$val"
    elif [ "$GPIO_MODE" = "gpiod_v2" ]; then
        # In libgpiod v2, gpioset exits immediately and releases the line unless kept alive.
        # Run gpioset in the background to hold the line state.
        if [ -n "${GPIOD_PIDS[$pin]}" ]; then
            kill "${GPIOD_PIDS[$pin]}" 2>/dev/null || true
        fi
        gpioset -c "$chip" "$pin=$val" &
        GPIOD_PIDS[$pin]=$!
    else
        echo "Error: Unknown GPIO mode or tools not installed." >&2
        return 1
    fi
}

gpio_get_value() {
    local chip="$1"
    local pin="$2"

    if [ "$GPIO_MODE" = "sysfs" ]; then
        cat "/sys/class/gpio/gpio${pin}/value"
    elif [ "$GPIO_MODE" = "gpiod_v1" ]; then
        gpioget "$chip" "$pin"
    elif [ "$GPIO_MODE" = "gpiod_v2" ]; then
        gpioget --numeric -c "$chip" "$pin"
    else
        echo "Error: Unknown GPIO mode or tools not installed." >&2
        return 1
    fi
}

gpio_set_value() {
    local chip="$1"
    local pin="$2"
    local val="$3"

    if [ "$GPIO_MODE" = "sysfs" ]; then
        echo "$val" > "/sys/class/gpio/gpio${pin}/value"
    elif [ "$GPIO_MODE" = "gpiod_v1" ]; then
        gpioset "$chip" "$pin=$val"
    elif [ "$GPIO_MODE" = "gpiod_v2" ]; then
        if [ -n "${GPIOD_PIDS[$pin]}" ]; then
            kill "${GPIOD_PIDS[$pin]}" 2>/dev/null || true
        fi
        gpioset -c "$chip" "$pin=$val" &
        GPIOD_PIDS[$pin]=$!
    else
        echo "Error: Unknown GPIO mode or tools not installed." >&2
        return 1
    fi
}

gpio_cleanup_pin() {
    local pin="$1"

    if [ "$GPIO_MODE" = "sysfs" ]; then
        if [ -d "/sys/class/gpio/gpio${pin}" ]; then
            echo "$pin" > /sys/class/gpio/unexport
        fi
    elif [ "$GPIO_MODE" = "gpiod_v2" ]; then
        if [ -n "${GPIOD_PIDS[$pin]}" ]; then
            kill "${GPIOD_PIDS[$pin]}" 2>/dev/null || true
            unset GPIOD_PIDS[$pin]
        fi
    fi
}

gpio_cleanup_all() {
    if [ "$GPIO_MODE" = "sysfs" ]; then
        for pin in "${!GPIOD_PIDS[@]}"; do
            gpio_cleanup_pin "$pin"
        done
    elif [ "$GPIO_MODE" = "gpiod_v2" ]; then
        for pin in "${!GPIOD_PIDS[@]}"; do
            if [ -n "${GPIOD_PIDS[$pin]}" ]; then
                kill "${GPIOD_PIDS[$pin]}" 2>/dev/null || true
            fi
        done
        GPIOD_PIDS=()
    fi
}
