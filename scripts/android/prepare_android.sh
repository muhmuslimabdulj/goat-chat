#!/bin/bash
set -e

# =============================================================================
# Prepare Android Platform
# Single Source of Truth - Called by BOTH local builds AND GitHub Actions
# =============================================================================

echo "Preparing Android platform..."

# Validate CAPACITOR_SERVER_URL
if [ -z "$CAPACITOR_SERVER_URL" ]; then
    # Try to load from .env file
    if [ -f ".env" ]; then
        export $(grep -v '^#' .env | grep CAPACITOR_SERVER_URL | xargs)
    fi
fi

if [ -z "$CAPACITOR_SERVER_URL" ]; then
    echo ""
    echo "ERROR: CAPACITOR_SERVER_URL is not set!"
    echo ""
    echo "Please set it in one of these ways:"
    echo "  1. Add to .env file: CAPACITOR_SERVER_URL=https://your-server.com"
    echo "  2. Export before running: export CAPACITOR_SERVER_URL=https://your-server.com"
    echo ""
    exit 1
fi

echo "Server URL: $CAPACITOR_SERVER_URL"

# 1. Sync Capacitor (copies web assets, updates capacitor.config.ts settings)
# Note: android/ folder is committed, so this just syncs web content
echo "Syncing Capacitor..."
npx cap sync android

# 2. Generate app icons and splash screens from resources/
# SKIPPED: Resources folder is missing in repo, and android/ folder already contains
# committed custom assets. Running generate would overwrite them with defaults.
# echo "Generating assets..."
# npx @capacitor/assets generate --android

echo "✓ Android preparation complete"
