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
APP_PORT=8080
GIN_MODE=debug # or release

# Database
DB_DRIVER=sqlite # or postgres
DB_NAME=gopickup.db
# If using Postgres:
# DB_HOST=localhost
# DB_USER=postgres
# DB_PASSWORD=password
# DB_PORT=5432

# Authentication
JWT_SECRET=your_super_secret_key

# Email (Plunk)
PLUNK_API_KEY=your_plunk_api_key

# CORS
CORS_ALLOW_ORIGINS=http://localhost:3000,https://myapp.com

# Flutterwave (Optional - Future)
# FLW_SECRET_KEY=...
# FLW_ENCRYPTION_KEY=...
```

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

## Hardening & Security (Phase 10)

- **Request Validation**: Strict validation on all inputs.
- **Structured Logging**: JSON-like logging with request latency and IDs.
- **Rate Limiting**: IP-based rate limiting on sensitive Auth endpoints.
- **Security Headers**: Standard security headers (HSTS, XSS protection, etc.).
- **CORS**: Strict CORS configuration via env vars.
- **Audit Logging**: Key actions (Order status, Bids, Product changes) are logged to the database.
- **CI/CD**: GitHub Actions workflow for linting and testing.
