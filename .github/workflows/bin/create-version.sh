#!/bin/bash

set -e

readonly PACKAGE_PATH="$1"
readonly CONTROL_PATH="$2"
readonly PACKAGE_VERSION="$3"
declare -a ARCH_LIST=("arm64" "armhf")

# Helper function for cross-platform in-place sed edits
sed_in_place() {
    if sed --version >/dev/null 2>&1; then
        # GNU sed (Linux)
        sed -i "$@"
    else
        # BSD/macOS sed
        sed -i '' "$@"
    fi
}

# Create the "versions" directory if it doesn't exist
mkdir -p "${PACKAGE_PATH}/versions"

# Ensure the package bin directory exists
mkdir -p "${PACKAGE_PATH}/x735-script-pkg/usr/bin"

for arch_type in "${ARCH_LIST[@]}"; do
    echo "Building for architecture: ${arch_type}..."

    # Set Go cross-compilation environment variables
    if [ "${arch_type}" = "arm64" ]; then
        export GOOS=linux
        export GOARCH=arm64
        export GOARM=""
    elif [ "${arch_type}" = "armhf" ]; then
        export GOOS=linux
        export GOARCH=arm
        export GOARM=7
    else
        echo "Unknown architecture: ${arch_type}"
        exit 1
    fi

    # Compile the Go application for the target architecture
    go build -ldflags="-s -w" -o "${PACKAGE_PATH}/x735-script-pkg/usr/bin/x735-daemon" "${PACKAGE_PATH}/main.go"

    # Create the x735off symlink pointing to x735-daemon in the package structure
    ln -sf x735-daemon "${PACKAGE_PATH}/x735-script-pkg/usr/bin/x735off"

    # Change the Architecture in the DEBIAN/control file
    sed_in_place "s/^Architecture:.*/Architecture: ${arch_type}/" "${CONTROL_PATH}"

    # Build the package if dpkg-deb is available
    if which dpkg-deb >/dev/null 2>&1; then
        dpkg-deb --build "${PACKAGE_PATH}/x735-script-pkg" "${PACKAGE_PATH}/versions/x735-script_${PACKAGE_VERSION}_${arch_type}.deb"
    else
        echo "Warning: 'dpkg-deb' not found. Skipping debian package compilation (this is expected on macOS without dpkg)."
    fi

    # Clean up binaries from staging folder before next loop / exit
    rm -f "${PACKAGE_PATH}/x735-script-pkg/usr/bin/x735-daemon"
    rm -f "${PACKAGE_PATH}/x735-script-pkg/usr/bin/x735off"
done

echo "Build complete."
