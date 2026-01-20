#!/bin/bash
set -e

# =============================================================================
# Run on Physical Device Script
# This script installs and launches the app on a connected Android device
# Run with: bash run_device.sh
# =============================================================================

echo "========================================"
echo "Deploy to Physical Android Device"
echo "========================================"

# Configuration
APK_PATH="android/app/build/outputs/apk/debug/app-debug.apk"
PACKAGE_NAME="com.goatchat.app"

# Set ANDROID_HOME
if [ -z "$ANDROID_HOME" ]; then
    export ANDROID_HOME="$HOME/android-sdk"
fi
export PATH="$ANDROID_HOME/platform-tools:$PATH"

# Force set Java 21 if it exists
if [ -d "/home/linuxbrew/.linuxbrew/opt/openjdk@21" ]; then
    export JAVA_HOME="/home/linuxbrew/.linuxbrew/opt/openjdk@21"
elif [ -d "/usr/lib/jvm/java-21-openjdk-amd64" ]; then
    export JAVA_HOME="/usr/lib/jvm/java-21-openjdk-amd64"
fi

if [ ! -z "$JAVA_HOME" ]; then
    export PATH="$JAVA_HOME/bin:$PATH"
fi

ADB="$ANDROID_HOME/platform-tools/adb"

# =============================================================================
# Step 1: Check for connected devices
# =============================================================================
echo ""
echo "[1/4] Checking for connected devices..."

# Try to connect via Windows host (for WSL2)
if [ -f /etc/resolv.conf ]; then
    WINDOWS_HOST=$(cat /etc/resolv.conf | grep nameserver | awk '{print $2}')
    if [ -n "$WINDOWS_HOST" ]; then
        echo "Attempting to connect via Windows ADB at $WINDOWS_HOST:5037..."
        export ADB_SERVER_SOCKET="tcp:$WINDOWS_HOST:5037"
    fi
fi

# List devices
DEVICES=$($ADB devices 2>/dev/null | grep -v "List" | grep -v "^$" | grep -v "emulator-" || true)

if [ -z "$DEVICES" ]; then
    echo ""
    echo "⚠ No physical device detected!"
    echo ""
    echo "To connect your Android device:"
    echo ""
    echo "Option 1: ADB over TCP (Recommended for WSL2)"
    echo "  1. Install ADB on Windows: https://developer.android.com/tools/releases/platform-tools"
    echo "  2. Connect phone via USB to Windows"
    echo "  3. On Windows CMD, run: adb tcpip 5555"
    echo "  4. Find phone IP in Settings > About > IP Address"
    echo "  5. Run: adb connect <phone-ip>:5555"
    echo "  6. Try this script again"
    echo ""
    echo "Option 2: Use usbipd-win"
    echo "  See: https://github.com/dorssel/usbipd-win"
    echo ""
    exit 1
fi

echo "✓ Found device(s):"
echo "$DEVICES"

# Get first non-emulator device
DEVICE_ID=$(echo "$DEVICES" | head -n1 | awk '{print $1}')
echo "Using device: $DEVICE_ID"

# =============================================================================
# Step 2: Check APK exists
# =============================================================================
echo ""
echo "[2/4] Checking APK..."

if [ ! -f "$APK_PATH" ]; then
    echo "⚠ APK not found at $APK_PATH"
    echo "Building APK first..."
    bash build_apk.sh
fi

echo "✓ APK ready: $APK_PATH"

# =============================================================================
# Step 3: Install APK
# =============================================================================
echo ""
echo "[3/4] Installing APK to device..."

$ADB -s "$DEVICE_ID" install -r "$APK_PATH"
echo "✓ APK installed"

# =============================================================================
# Step 4: Launch App
# =============================================================================
echo ""
echo "[4/4] Launching app..."

$ADB -s "$DEVICE_ID" shell monkey -p "$PACKAGE_NAME" -c android.intent.category.LAUNCHER 1 > /dev/null 2>&1
echo "✓ App launched!"

echo ""
echo "========================================"
echo "✓ Done! Check your phone."
echo "========================================"
echo ""
