#!/bin/sh
set -e

echo "[ENTRYPOINT] Current user: $(whoami)"
echo "[ENTRYPOINT] Looking for migrate binary..."
if [ -f "./migrate" ]; then
    echo "[ENTRYPOINT] Found migrate binary."
else
    echo "[ENTRYPOINT] ERROR: migrate binary not found!"
    exit 1
fi

echo "[ENTRYPOINT] Running database migrations..."
./migrate

echo "[ENTRYPOINT] Looking for server binary..."
if [ -f "./server" ]; then
    echo "[ENTRYPOINT] Found server binary."
else
    echo "[ENTRYPOINT] ERROR: server binary not found!"
    exit 1
fi

echo "[ENTRYPOINT] Starting application server..."
exec ./server
