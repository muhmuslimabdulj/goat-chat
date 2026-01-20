#!/bin/bash
set -e

# =============================================================================
# Android SDK Setup Script for WSL2/Linux
# This script installs Android SDK, emulator, and all required dependencies
# Run with: bash setup_android.sh
# =============================================================================

echo "========================================"
echo "Android SDK Setup for WSL2/Linux"
echo "========================================"

# Configuration
SDK_DIR="$HOME/android-sdk"
CMD_TOOLS_URL="https://dl.google.com/android/repository/commandlinetools-linux-11076708_latest.zip"
ANDROID_API="34"
AVD_NAME="pixel_5_api_${ANDROID_API}"
SYSTEM_IMAGE="system-images;android-${ANDROID_API};google_apis;x86_64"

# =============================================================================
# Step 1: Install System Dependencies
# =============================================================================
echo ""
echo "[1/6] Installing system dependencies..."

# Check if running as root for apt
if command -v sudo &> /dev/null; then
    SUDO="sudo"
else
    SUDO=""
fi

# Install required packages
$SUDO apt-get update -qq
$SUDO apt-get install -y -qq \
    openjdk-21-jdk \
    wget \
    unzip \
    libxcb-cursor0 \
    libxcb-icccm4 \
    libxcb-keysyms1 \
    libxcb-image0 \
    libxcb-shm0 \
    libxcb-sync1 \
    libxcb-xfixes0 \
    libxcb-shape0 \
    libxcb-randr0 \
    libxcb-render-util0 \
    libxcb-xinerama0 \
    libxkbcommon-x11-0 \
    libpulse0 \
    libgl1 \
    libnss3 \
    libxcomposite1 \
    libxcursor1 \
    libxi6 \
    libxtst6 \
    libasound2t64 \
    libsm6 \
    libxkbfile1 2>/dev/null || \
$SUDO apt-get install -y -qq \
    openjdk-21-jdk \
    wget \
    unzip \
    libxcb-cursor0 \
    libxcb-icccm4 \
    libxcb-keysyms1 \
    libxcb-image0 \
    libxcb-shm0 \
    libxcb-sync1 \
    libxcb-xfixes0 \
    libxcb-shape0 \
    libxcb-randr0 \
    libxcb-render-util0 \
    libxcb-xinerama0 \
    libxkbcommon-x11-0 \
    libpulse0 \
    libgl1 \
    libnss3 \
    libxcomposite1 \
    libxcursor1 \
    libxi6 \
    libxtst6 \
    libasound2 \
    libsm6 \
    libxkbfile1

echo "✓ System dependencies installed"

# =============================================================================
# Step 2: Detect Java
# =============================================================================
echo ""
echo "[2/6] Detecting Java..."

# Try to find Java (Favor 21)
if [ -d "/home/linuxbrew/.linuxbrew/opt/openjdk@21" ]; then
    export JAVA_HOME="/home/linuxbrew/.linuxbrew/opt/openjdk@21"
    echo "✓ Found Homebrew Java 21: $JAVA_HOME"
elif [ -d "/usr/lib/jvm/java-21-openjdk-amd64" ]; then
    export JAVA_HOME="/usr/lib/jvm/java-21-openjdk-amd64"
    echo "✓ Found system Java 21: $JAVA_HOME"
elif [ -d "/home/linuxbrew/.linuxbrew/opt/openjdk@17" ]; then
    export JAVA_HOME="/home/linuxbrew/.linuxbrew/opt/openjdk@17"
    echo "✓ Found Homebrew Java 17: $JAVA_HOME"
elif [ -d "/usr/lib/jvm/java-17-openjdk-amd64" ]; then
    export JAVA_HOME="/usr/lib/jvm/java-17-openjdk-amd64"
    echo "✓ Found system Java 17: $JAVA_HOME"
elif [ -n "$JAVA_HOME" ] && [ -d "$JAVA_HOME" ]; then
    echo "✓ Using existing JAVA_HOME: $JAVA_HOME"
else
    echo "Error: Java 21 or 17 not found. Please install OpenJDK."
    exit 1
fi

# =============================================================================
# Step 3: Setup Android SDK
# =============================================================================
echo ""
echo "[3/6] Setting up Android SDK..."

export ANDROID_HOME="$SDK_DIR"
export PATH="$ANDROID_HOME/cmdline-tools/latest/bin:$ANDROID_HOME/emulator:$ANDROID_HOME/platform-tools:$PATH"

mkdir -p "$SDK_DIR/cmdline-tools"

if [ ! -d "$SDK_DIR/cmdline-tools/latest" ]; then
    echo "Downloading Android Command Line Tools..."
    wget -q --show-progress "$CMD_TOOLS_URL" -O /tmp/cmdline-tools.zip
    
    unzip -q /tmp/cmdline-tools.zip -d /tmp/android-temp
    mv /tmp/android-temp/cmdline-tools "$SDK_DIR/cmdline-tools/latest"
    
    rm -f /tmp/cmdline-tools.zip
    rm -rf /tmp/android-temp
    echo "✓ Command Line Tools installed"
else
    echo "✓ Command Line Tools already installed"
fi

# =============================================================================
# Step 4: Install SDK Packages
# =============================================================================
echo ""
echo "[4/6] Installing SDK packages (this may take a while)..."

yes | "$SDK_DIR/cmdline-tools/latest/bin/sdkmanager" --licenses > /dev/null 2>&1 || true

"$SDK_DIR/cmdline-tools/latest/bin/sdkmanager" \
    "platform-tools" \
    "platforms;android-${ANDROID_API}" \
    "build-tools;${ANDROID_API}.0.0" \
    "emulator" \
    "$SYSTEM_IMAGE"

echo "✓ SDK packages installed"

# =============================================================================
# Step 5: Create AVD
# =============================================================================
echo ""
echo "[5/6] Creating Android Virtual Device..."

if "$SDK_DIR/cmdline-tools/latest/bin/avdmanager" list avd 2>/dev/null | grep -q "$AVD_NAME"; then
    echo "✓ AVD '$AVD_NAME' already exists"
else
    echo "no" | "$SDK_DIR/cmdline-tools/latest/bin/avdmanager" create avd \
        -n "$AVD_NAME" \
        -k "$SYSTEM_IMAGE" \
        --device "pixel_5" \
        --force
    echo "✓ AVD '$AVD_NAME' created"
fi

# =============================================================================
# Step 6: Fix KVM Permissions
# =============================================================================
echo ""
echo "[6/6] Fixing KVM permissions..."

if [ -e /dev/kvm ]; then
    # Add user to kvm group
    $SUDO usermod -aG kvm "$USER" 2>/dev/null || true
    
    # Set permissions (immediate fix without re-login)
    $SUDO chmod 666 /dev/kvm 2>/dev/null || true
    
    echo "✓ KVM permissions configured"
else
    echo "⚠ KVM not available. Emulator will use software rendering (slower)."
fi

# =============================================================================
# Step 7: Update Shell Configuration
# =============================================================================
echo ""
echo "Updating shell configuration..."

RC_FILE="$HOME/.bashrc"
if ! grep -q "ANDROID_HOME" "$RC_FILE" 2>/dev/null; then
    cat >> "$RC_FILE" << EOF

# Android SDK (added by setup_android.sh)
export JAVA_HOME="$JAVA_HOME"
export ANDROID_HOME="$SDK_DIR"
export PATH="\$ANDROID_HOME/cmdline-tools/latest/bin:\$ANDROID_HOME/emulator:\$ANDROID_HOME/platform-tools:\$PATH"
EOF
    echo "✓ Environment variables added to $RC_FILE"
else
    echo "✓ Environment variables already in $RC_FILE"
fi

# =============================================================================
# Done!
# =============================================================================
echo ""
echo "========================================"
echo "✓ Setup Complete!"
echo "========================================"
echo ""
echo "Next steps:"
echo "  1. Restart your terminal OR run: source ~/.bashrc"
echo "  2. Build the APK: ./build_apk.sh"
echo "  3. Run the emulator: ./run_emulator.sh"
echo ""
echo "If emulator window doesn't appear, restart WSL:"
echo "  (in PowerShell) wsl --shutdown"
echo ""
