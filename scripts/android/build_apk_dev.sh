#!/bin/bash
set -e

# =============================================================================
# Build APK for Development (localhost)
# This script builds the APK pointing to a local development server
# Run with: bash build_apk_dev.sh [optional-port]
# =============================================================================

# Default port
PORT="${1:-8080}"

echo "========================================"
echo "Building Development APK"
echo "========================================"
echo ""
echo "Target: Local development server on port $PORT"
echo ""

# Detect IP for different targets
PC_IP=$(hostname -I | awk '{print $1}')

echo "Server URLs that will work:"
echo "  - Emulator:  http://10.0.2.2:$PORT"
echo "  - HP Asli:   http://$PC_IP:$PORT"
echo ""

# Ask user which target
echo "Pilih target:"
echo "  1) Emulator (10.0.2.2)"
echo "  2) HP Asli (Input Manual IP Windows)"
echo ""
read -p "Pilihan [1/2]: " choice

case $choice in
    2)
        read -p "Masukkan IP Windows (cek ipconfig, misal 192.168.1.5): " MANUAL_IP
        export CAPACITOR_SERVER_URL="http://$MANUAL_IP:$PORT"
        echo "→ Menggunakan: $CAPACITOR_SERVER_URL"
        ;;
    *)
        export CAPACITOR_SERVER_URL="http://10.0.2.2:$PORT"
        echo "→ Menggunakan: $CAPACITOR_SERVER_URL"
        ;;
esac

echo ""

# Run the main build script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
bash "$SCRIPT_DIR/build_apk.sh"

echo ""
echo "========================================"
echo "⚠ PENTING: Pastikan server development berjalan!"
echo "========================================"
echo ""
echo "Jalankan server Anda di port $PORT sebelum membuka app."
echo "Contoh: go run cmd/server/main.go"
echo ""
