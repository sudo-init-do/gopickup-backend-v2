#!/bin/sh
set -e

echo "Running database migrations..."
./migrate

echo "Starting application server..."
exec ./server
