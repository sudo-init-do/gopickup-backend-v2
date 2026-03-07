#!/bin/sh
set -e

echo "Running database migrations..."
./migrate

echo "Seeding database..."
# Run seeder with ALLOW_SEED=true to bypass production safety check
ALLOW_SEED=true ./seed -seed

echo "Starting application server..."
exec ./server
