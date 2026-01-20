#!/bin/bash
set -e

# =============================================================================
# Run Emulator Script
# This script starts the Android emulator, installs the APK, and launches the app
# Run with: bash run_emulator.sh
# =============================================================================

echo "========================================"
echo "Running Android Emulator"
echo "========================================"

# Configuration
AVD_NAME="pixel_5_api_34"
SYSTEM_IMAGE="system-images;android-34;google_apis;x86_64"
APK_PATH="android/app/build/outputs/apk/debug/app-debug.apk"
PACKAGE_NAME="com.goatchat.app"

# =============================================================================
# Step 1: Check Prerequisites
# =============================================================================
echo ""
echo "[1/5] Checking prerequisites..."

# Set ANDROID_HOME
if [ -z "$ANDROID_HOME" ]; then
    export ANDROID_HOME="$HOME/android-sdk"
fi

# Force set Java 21 if it exists
if [ -d "/home/linuxbrew/.linuxbrew/opt/openjdk@21" ]; then
    export JAVA_HOME="/home/linuxbrew/.linuxbrew/opt/openjdk@21"
elif [ -d "/usr/lib/jvm/java-21-openjdk-amd64" ]; then
    export JAVA_HOME="/usr/lib/jvm/java-21-openjdk-amd64"
fi

if [ ! -z "$JAVA_HOME" ]; then
    export PATH="$JAVA_HOME/bin:$PATH"
fi

export PATH="$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/emulator:$ANDROID_HOME/platform-tools:$PATH"

# Check if SDK is installed
if [ ! -d "$ANDROID_HOME/emulator" ]; then
    echo "Error: Android SDK not found at $ANDROID_HOME"
    echo "Please run ./setup_android.sh first."
    exit 1
fi
echo "✓ Android SDK found"

# Check KVM
if [ -e /dev/kvm ]; then
    if [ ! -r /dev/kvm ] || [ ! -w /dev/kvm ]; then
        echo "⚠ KVM permission issue. Attempting to fix..."
        sudo chmod 666 /dev/kvm 2>/dev/null || true
    fi
fi

# =============================================================================
# Step 2: Create AVD if needed
# =============================================================================
echo ""
echo "[2/5] Checking AVD..."

if "$ANDROID_HOME/cmdline-tools/latest/bin/avdmanager" list avd 2>/dev/null | grep -q "$AVD_NAME"; then
    echo "✓ AVD '$AVD_NAME' exists"
else
    echo "Creating AVD '$AVD_NAME'..."
    echo "no" | "$ANDROID_HOME/cmdline-tools/latest/bin/avdmanager" create avd \
        -n "$AVD_NAME" \
        -k "$SYSTEM_IMAGE" \
        --device "pixel_5" \
        --force
    echo "✓ AVD created"
fi

# =============================================================================
# Step 3: Start Emulator
# =============================================================================
echo ""
echo "[3/5] Starting emulator..."

# Check if already running
if "$ANDROID_HOME/platform-tools/adb" devices 2>/dev/null | grep -q "emulator-"; then
    echo "✓ Emulator is already running"
else
    # Clear any stale locks
    rm -f "$HOME/.android/avd/${AVD_NAME}.avd"/*.lock 2>/dev/null || true
    
    # Start emulator with software rendering (more compatible with WSL2)
    nohup "$ANDROID_HOME/emulator/emulator" \
        -avd "$AVD_NAME" \
        -netdelay none \
        -netspeed full \
        -gpu swiftshader_indirect \
        -no-snapshot-load \
        > emulator.log 2>&1 &
    
    echo "Waiting for emulator to boot..."
    echo "(You can check emulator.log for details)"
    
    # Wait for device
    "$ANDROID_HOME/platform-tools/adb" wait-for-device
    echo "✓ Emulator device found"
    
    # Wait for boot completion
    echo "Waiting for system boot..."
    while [ "$("$ANDROID_HOME/platform-tools/adb" shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')" != "1" ]; do
        sleep 2
        echo -n "."
    done
    echo ""
    echo "✓ Emulator is ready!"
fi

# =============================================================================
# Step 4: Install APK
# =============================================================================
echo ""
echo "[4/5] Installing APK..."

# Get emulator device ID
EMU_ID=$($ANDROID_HOME/platform-tools/adb devices | grep "emulator-" | head -n1 | awk '{print $1}')

if [ -z "$EMU_ID" ]; then
    echo "⚠ Emulator not found in adb devices"
    exit 1
fi

if [ -f "$APK_PATH" ]; then
    "$ANDROID_HOME/platform-tools/adb" -s "$EMU_ID" install -r "$APK_PATH"
    echo "✓ APK installed"
else
    echo "⚠ APK not found at $APK_PATH"
    echo "  Run ./build_apk.sh first to build the APK."
fi

# =============================================================================
# Step 5: Launch App
# =============================================================================
echo ""
echo "[5/5] Launching app..."

"$ANDROID_HOME/platform-tools/adb" -s "$EMU_ID" shell monkey -p "$PACKAGE_NAME" -c android.intent.category.LAUNCHER 1 > /dev/null 2>&1
echo "✓ App launched!"

echo ""
echo "========================================"
echo "✓ Done! Check the emulator window."
echo "========================================"
echo ""
echo "Tips:"
echo "  - If window doesn't appear, restart WSL: wsl --shutdown (in PowerShell)"
echo "  - Logs are in: emulator.log"
echo ""
