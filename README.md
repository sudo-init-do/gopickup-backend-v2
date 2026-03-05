# GoPickup Backend

## Overview

GoPickup is a backend service for a pickup and delivery application, built with Go (Golang) and the Gin web framework. It supports multiple user roles (Client, Driver, Vendor), order management, bidding system, real-time chat, and more.

## Prerequisites

- Go 1.23+
- PostgreSQL or SQLite
- Redis (Optional, for future rate limiting scaling)

## Environment Variables

Create a `.env` file in the root directory:

```env
# Server
APP_ENV=development # or production, staging
APP_PORT=8080
GIN_MODE=debug # or release

# Database
DB_DRIVER=sqlite # or postgres
# If using Postgres (Recommended for Prod):
# DATABASE_URL=postgres://user:pass@host:5432/dbname?sslmode=disable
# OR separate fields:
# DB_HOST=localhost
# DB_USER=postgres
# DB_PASSWORD=password
# DB_PORT=5432
# DB_NAME=gopickup

# Authentication
JWT_SECRET=your_super_secret_key

# Email (Plunk)
PLUNK_API_KEY=your_plunk_api_key
PLUNK_FROM_EMAIL=hello@gopickup.com
PLUNK_FROM_NAME=GoPickup Team

# CORS
CORS_ALLOW_ORIGINS=http://localhost:3000,https://myapp.com,*

# Migrations
MIGRATE_ON_START=true # Set to false in production if running migrations separately

# Monitoring
ENABLE_METRICS=true # Optional: Enable /api/v1/metrics endpoint
```

## Documentation

- [Deployment Runbook](docs/RUNBOOK.md)
- [Backups & Restore](docs/BACKUPS.md)
- [Monitoring & Alerts](docs/MONITORING.md)
- [Security Checklist](docs/SECURITY_CHECKLIST.md)
- [Staging Setup](docs/STAGING_SETUP.md)
- [Smoke Tests](docs/SMOKE_TEST.md)

## Running Locally

1.  **Clone the repository:**
    ```bash
    git clone <repo_url>
    cd gopickup-backend
    ```

2.  **Install dependencies:**
    ```bash
    go mod download
    ```

3.  **Run the application:**
    ```bash
    go run cmd/server/main.go
    ```

## Running Tests

To run all tests:

```bash
go test -v ./...
```

## Deployment

### Docker (Preferred)

1.  **Build the image:**
    ```bash
    docker build -t gopickup-backend:latest .
    ```

2.  **Run with Docker Compose (Production):**
    ```bash
    docker-compose -f docker-compose.prod.yml up -d
    ```

### Manual VPS Deployment

1.  Build the binary:
    ```bash
    CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server
    CGO_ENABLED=0 GOOS=linux go build -o migrate ./cmd/migrate
    ```
2.  Copy `server` and `migrate` to your server.
3.  Set environment variables (see above).
4.  Run migrations: `./migrate`
5.  Start server: `./server`
6.  Use Nginx as a reverse proxy (see `nginx/nginx.conf` for example).

### Database Migrations

- **Development**: Migrations run automatically on startup by default.
- **Production**:
    - Set `MIGRATE_ON_START=false` to disable auto-migration.
    - Run the migration tool explicitly:
      ```bash
      # Using Docker
      docker-compose -f docker-compose.prod.yml run --rm migrate
      # Or manual binary
      ./migrate
      ```

## Smoke Tests

A comprehensive checklist of curl commands to verify the deployment is available in [docs/SMOKE_TEST.md](docs/SMOKE_TEST.md).

## Developer Tools

### Seed Admin User
Create an admin user for testing:
```bash
go run cmd/seed/main.go -admin -email=admin@example.com -password=SecretPass123!
```

### Verify User (Bypass Email OTP)
Mark a user as verified in development:
```bash
go run cmd/seed/main.go -verify=user@example.com
```

## API Overview

### Public Endpoints
- `GET /api/v1/health` - Health check
- `GET /api/v1/ready` - Readiness check (DB connection)
- `GET /api/v1/ws` - WebSocket connection
- `GET /api/v1/products` - List products
- `GET /api/v1/products/:id` - Get product details
- `GET /api/v1/vendors` - List vendors

### Auth Endpoints (Rate Limited)
- `POST /api/v1/auth/register` - Register user
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/verify-otp` - Verify OTP

### Protected Endpoints (Requires Bearer Token)
- **Chat**: Initiate, List, Get Messages, Mark Read
- **Orders**: Create, List, Get Details, Accept Bid
- **Profile**: Create/Update profiles for Client, Driver, Vendor
- **Driver**: Update Location, Get Jobs, Place Bid
- **Vendor**: Manage Products, Update Order Status
