#!/bin/bash
set -e

# Load environment variables if .env exists
if [ -f .env ]; then
  export $(grep -v '^#' .env | xargs)
fi

echo "Running migrations..."
# Assuming we are in the project root and using docker-compose
docker-compose -f docker-compose.prod.yml run --rm migrate
echo "Migration completed."
