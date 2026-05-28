#!/bin/bash

set -e

readonly PACKAGE_PATH="$1"
readonly CONTROL_PATH="$2"
readonly PACKAGE_VERSION="$3"
declare -a ARCH_LIST=("arm64" "armhf")

# Create the "versions" directory if it doesn't exist
mkdir -p "${PACKAGE_PATH}/versions"

for arch_type in "${ARCH_LIST[@]}"; do
    # First change the Architecture in the DEBIAN/control file
    sed -i "/^Architecture:.*/ c Architecture: ${arch_type}" "${CONTROL_PATH}"

    # Build the package natively on the runner
    dpkg-deb --build "${PACKAGE_PATH}/x735-script-pkg" "${PACKAGE_PATH}/versions/x735-script_${PACKAGE_VERSION}_${arch_type}.deb"
done
