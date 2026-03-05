# Staging Environment Setup Guide

This guide explains how to set up the GoPickup backend in a staging environment for verification and client handoff.

## 1. Environment Variables

Create a `.env` file (or configure your cloud provider's environment variables) with the following:

```env
# Server
APP_ENV=staging
APP_PORT=8080
GIN_MODE=release

# Database
# Use a separate database for staging!
DATABASE_URL=postgres://user:password@staging-db-host:5432/gopickup_staging?sslmode=require

# Authentication
JWT_SECRET=staging_secret_key_change_me

# Email (Plunk) - Use a testing API key if available
PLUNK_API_KEY=your_plunk_api_key
PLUNK_FROM_EMAIL=staging@gopickup.com
PLUNK_FROM_NAME=GoPickup Staging

# CORS
# whitelist the staging frontend URL
CORS_ALLOW_ORIGINS=https://staging.gopickup.com,http://localhost:3000

# Seeding (Enable only if you need to reset/seed data)
ALLOW_SEED=true

# Migrations
MIGRATE_ON_START=true
```

## 2. Database Migrations

In staging, you can allow the application to run migrations on start by setting `MIGRATE_ON_START=true`.

Alternatively, run them manually via Docker:
```bash
docker-compose -f docker-compose.prod.yml run --rm migrate
```

## 3. Creating Test Accounts (Seeding)

We have a helper tool to create verified test accounts for all roles (Admin, Client, Driver, Vendor).

**WARNING**: This will insert data into your database. Ensure you are connected to the STAGING database.

To run the seeder (requires `ALLOW_SEED=true` env var):

```bash
# If running locally against staging DB
ALLOW_SEED=true go run cmd/seed/main.go -seed

# If running via Docker
docker-compose -f docker-compose.prod.yml run --rm -e ALLOW_SEED=true server /app/server -seed
# Note: You might need to adjust the entrypoint or command depending on how the image is built.
# The image default CMD is "./server". We can override it to run the seed command if we built a combined binary or use `go run` if source is available (not recommended for prod image).
# Better approach for compiled image:
# The `cmd/seed` is not currently built into the main server binary.
# You should build it separately or run it from source if available.
```

**Recommended Staging Workflow for Seeding:**
1.  Deploy the code.
2.  SSH into the instance or use a "run-task" feature.
3.  If the source code is not available, you may need to build the seed binary locally and scp it, or include it in the Docker image.
    *   *Note: The current Dockerfile copies `server` and `migrate`. You may want to update the Dockerfile to include `seed` for staging convenience.*

### Default Seeded Accounts
All passwords are: `Password123!`

- **Admin**: `admin@test.com`
- **Client**: `client@test.com`
- **Driver**: `driver@test.com` (Approved, Van)
- **Vendor**: `vendor@test.com` (Approved, Retail)

## 4. CORS Settings

Ensure `CORS_ALLOW_ORIGINS` includes your staging frontend URL.
Example: `CORS_ALLOW_ORIGINS=https://staging-frontend.vercel.app,http://localhost:3000`

## 5. WebSocket Connection

The WebSocket endpoint is available at `/api/v1/ws`.

**Connection URL**: `wss://your-staging-domain.com/api/v1/ws` (or `ws://` if no SSL)

**Authentication**:
You must provide the JWT token in the `Authorization` header during the handshake.
Since standard WebSocket APIs in browsers don't support custom headers easily, you might need to use a query parameter if the backend supports it (currently backend expects Header).
*Correction*: The current backend implementation strictly looks for `Authorization` header (Bearer token) or `Sec-WebSocket-Protocol`.
Ensure your client library supports sending auth headers.

## 6. Debugging & Safety

- **Logs**: In staging (`GIN_MODE=release`), logs are JSON formatted.
- **Health Check**: `GET /api/v1/health` should return `200 OK`.
- **Readiness**: `GET /api/v1/ready` checks DB connection.
