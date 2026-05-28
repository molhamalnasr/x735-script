#!/usr/bin/env bash

set -e

readonly SHUTDOWN=5
readonly BOOT=12
readonly GPIO_CHIP=0
readonly REBOOT_PULSE_MINIMUM="200"
readonly REBOOT_PULSE_MAXIMUM="600"

# Source the GPIO helper script
readonly HELPER_PATH="/usr/lib/x735-script/scripts/gpio-helper.sh"
if [ -f "$HELPER_PATH" ]; then
    source "$HELPER_PATH"
else
    # Fallback if running from source folder during development/testing
    source "$(dirname "${BASH_SOURCE[0]}")/gpio-helper.sh"
fi

terminate() {
  local -r msg=$1
  local -r err_code=${2:-150}
  echo "${msg}" >&2
  exit "${err_code}"
}

gpio_cleanup() {
  gpio_cleanup_all
  terminate "Unexpected exit..."
}

init_pins() {
  gpio_init_input "$GPIO_CHIP" "$SHUTDOWN"
  gpio_init_output "$GPIO_CHIP" "$BOOT" 1
}

echo "The x735-script is listening to the shutdown button clicks..."

# Main method
__main__() {
  # Handle exit and interrupt signals to clean up GPIO
  trap gpio_cleanup EXIT SIGINT SIGTERM

  init_pins

  while true; do
    local shutdown_signal="$(gpio_get_value "$GPIO_CHIP" "$SHUTDOWN")"
    if [ "$shutdown_signal" = "0" ]; then
      sleep 0.2
    else
      local pulse_start=$(($(date +%s%N | cut -b1-13)))
      while [ "$shutdown_signal" = "1" ]; do
        sleep 0.02
        local current_time=$(($(date +%s%N | cut -b1-13)))
        local elapsed_time=$((current_time - pulse_start))
        if [ "$elapsed_time" -gt "$REBOOT_PULSE_MAXIMUM" ]; then
          echo "Your device is shutting down (GPIO $SHUTDOWN), halting RPi ..."
          poweroff
          exit
        fi
        shutdown_signal="$(gpio_get_value "$GPIO_CHIP" "$SHUTDOWN")"
      done
      if [ "$elapsed_time" -gt "$REBOOT_PULSE_MINIMUM" ]; then
        echo "Your device is rebooting (GPIO $SHUTDOWN), recycling RPi ..."
        reboot
        exit
      fi
    fi
  done
}

__main__
