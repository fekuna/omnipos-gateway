#!/bin/sh
# Development entrypoint

echo "Starting gateway with Air..."
cd /app/omnipos-gateway
exec air -c .air.toml
