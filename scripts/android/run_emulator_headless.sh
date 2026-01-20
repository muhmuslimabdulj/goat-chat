#!/bin/bash
set -e

# Configuration
AVD_NAME="pixel_5_api_34"

# Ensure Environment Variables are loaded
if [ -z "$ANDROID_HOME" ]; then
    export ANDROID_HOME="$HOME/android-sdk"
    
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
fi

# Check if Emulator is running
if $ANDROID_HOME/platform-tools/adb devices | grep -q "emulator-"; then
    echo "Emulator is already running."
else
    echo "Starting Emulator '$AVD_NAME' in HEADLESS mode..."
    # -no-window: No GUI
    # -no-audio: No Audio
    nohup $ANDROID_HOME/emulator/emulator -avd "$AVD_NAME" -netdelay none -netspeed full -no-window -no-audio > emulator_headless.log 2>&1 &
    
    echo "Emulator started in background. Waiting for boot..."
    echo "Logs: emulator_headless.log"
    
    $ANDROID_HOME/platform-tools/adb wait-for-device
    echo "Device found! Waiting for system boot..."
    
    # Wait loop
    while [ "$($ANDROID_HOME/platform-tools/adb shell getprop sys.boot_completed 2>/dev/null | tr -d '\r')" != "1" ]; do
        sleep 2
        echo -n "."
    done
    echo ""
    echo "Emulator is globally ready!"
fi

# Install APK
APK_PATH="android/app/build/outputs/apk/debug/app-debug.apk"
if [ -f "$APK_PATH" ]; then
    echo "Installing APK..."
    $ANDROID_HOME/platform-tools/adb install -r "$APK_PATH"
    echo "App installed successfully!"
    
    PACKAGE_NAME="com.goatchat.app"
    echo "Launching $PACKAGE_NAME..."
    $ANDROID_HOME/platform-tools/adb shell monkey -p "$PACKAGE_NAME" -c android.intent.category.LAUNCHER 1
    echo "App launched! (Headless)"
else
    echo "APK not found at $APK_PATH"
fi
