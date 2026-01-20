#!/bin/bash
set -e

# =============================================================================
# Build APK Script
# This script syncs Capacitor, generates assets, and builds the Android APK
# Run with: bash build_apk.sh
# =============================================================================

echo "========================================"
echo "Building Android APK"
echo "========================================"

# =============================================================================
# Step 1: Detect Node.js
# =============================================================================
echo ""
echo "[1/4] Detecting Node.js..."

# Try multiple sources for Node
if command -v node &> /dev/null; then
    echo "✓ Node found: $(node --version)"
elif [ -s "$HOME/.nvm/nvm.sh" ]; then
    echo "Loading NVM..."
    export NVM_DIR="$HOME/.nvm"
    source "$NVM_DIR/nvm.sh"
    echo "✓ Node loaded via NVM: $(node --version)"
elif [ -s "$HOME/.local/share/pnpm/pnpm" ]; then
    export PNPM_HOME="$HOME/.local/share/pnpm"
    export PATH="$PNPM_HOME:$PATH"
    echo "✓ Using pnpm"
else
    echo "Error: Node.js not found!"
    echo "Please install Node.js via:"
    echo "  - nvm: https://github.com/nvm-sh/nvm"
    echo "  - Or: sudo apt install nodejs npm"
    exit 1
fi

# Ensure npm/npx is available
if ! command -v npx &> /dev/null; then
    echo "Error: npx not found. Please install npm."
    exit 1
fi

# =============================================================================
# Step 2: Install Dependencies (if needed)
# =============================================================================
echo ""
echo "[2/4] Checking dependencies..."

if [ ! -d "node_modules" ]; then
    echo "Installing npm dependencies..."
    npm install
fi
echo "✓ Dependencies ready"

# =============================================================================
# Step 3: Prepare Android (Single Source of Truth)
# =============================================================================
echo ""
echo "[3/4] Preparing Android..."

bash scripts/android/prepare_android.sh

# =============================================================================
# Step 4: Build APK
# =============================================================================
echo ""
echo "[4/4] Building APK with Gradle..."

# Ensure ANDROID_HOME is set
if [ -z "$ANDROID_HOME" ]; then
    export ANDROID_HOME="$HOME/android-sdk"
fi

# Force set Java 21 if it exists (favors Brew then System)
if [ -d "/home/linuxbrew/.linuxbrew/opt/openjdk@21" ]; then
    export JAVA_HOME="/home/linuxbrew/.linuxbrew/opt/openjdk@21"
elif [ -d "/usr/lib/jvm/java-21-openjdk-amd64" ]; then
    export JAVA_HOME="/usr/lib/jvm/java-21-openjdk-amd64"
elif [ -z "$JAVA_HOME" ]; then
    # Fallback to 17 only if JAVA_HOME not already set
    if [ -d "/home/linuxbrew/.linuxbrew/opt/openjdk@17" ]; then
        export JAVA_HOME="/home/linuxbrew/.linuxbrew/opt/openjdk@17"
    elif [ -d "/usr/lib/jvm/java-17-openjdk-amd64" ]; then
        export JAVA_HOME="/usr/lib/jvm/java-17-openjdk-amd64"
    fi
fi

# Ensure Java is in PATH and export it
if [ ! -z "$JAVA_HOME" ]; then
    export PATH="$JAVA_HOME/bin:$PATH"
    echo "Using JAVA_HOME: $JAVA_HOME"
    java -version
fi

cd android
./gradlew assembleDebug
cd ..

APK_PATH="android/app/build/outputs/apk/debug/app-debug.apk"

echo ""
echo "========================================"
echo "✓ Build Complete!"
echo "========================================"
echo ""
echo "APK Location: $APK_PATH"
echo ""
echo "Next: Run the emulator with ./run_emulator.sh"
echo ""
