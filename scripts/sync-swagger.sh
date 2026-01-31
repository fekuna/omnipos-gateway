#!/bin/bash
set -e

SPECS_DIR="internal/swagger/specs"

echo "🔍 Syncing swagger specs for production build..."

# Try local proto repo first (for local builds)
PROTO_OPENAPI="../omnipos-proto/openapi"
if [ -d "$PROTO_OPENAPI" ]; then
    echo "📋 Found local proto directory: $PROTO_OPENAPI"
    mkdir -p "$SPECS_DIR"
    find "$PROTO_OPENAPI" -name "*.swagger.json" -exec cp -v {} "$SPECS_DIR/" \;
    echo "✅ Synced from local proto repository"
    exit 0
fi

# Fallback: Get from Go module (for CI/CD)
echo "📦 Fetching from Go module dependency..."
PROTO_PKG=$(go list -m -f '{{.Dir}}' github.com/fekuna/omnipos-proto 2>/dev/null || echo "")

if [ -n "$PROTO_PKG" ] && [ -d "$PROTO_PKG/openapi" ]; then
    echo "📋 Found proto package at: $PROTO_PKG"
    mkdir -p "$SPECS_DIR"
    find "$PROTO_PKG/openapi" -name "*.swagger.json" -exec cp -v {} "$SPECS_DIR/" \;
    echo "✅ Synced from Go module"
    exit 0
fi

echo "❌ Could not find swagger specs!"
echo "ℹ️  Make sure either:"
echo "   1. ../omnipos-proto exists (local development)"
echo "   2. github.com/fekuna/omnipos-proto is in go.mod (production)"
exit 1
