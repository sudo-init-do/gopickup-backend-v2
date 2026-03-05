# Deployment Runbook

This guide outlines the standard operating procedure for deploying GoPickup to production.

## 1. Prerequisites

- **Environment**: Ensure `.env` is set to `APP_ENV=production`.
- **Database**: Ensure database is reachable and migrations are prepared.
- **Access**: SSH access to production server or CI/CD pipeline access.

## 2. Deployment Steps (Docker)

### Step 1: Pull Latest Image
```bash
docker-compose -f docker-compose.prod.yml pull
```

### Step 2: Run Database Migrations (Pre-Deploy)
This ensures the database schema is compatible with the new code.
```bash
docker-compose -f docker-compose.prod.yml run --rm migrate
```
*Note: If migration fails, DO NOT proceed. Investigate and rollback if necessary.*

### Step 3: Restart Application (Rolling Update)
If using Docker Swarm or Kubernetes, use rolling updates. For single-host Docker Compose:
```bash
docker-compose -f docker-compose.prod.yml up -d --no-deps --build server
```
*This restarts only the server container with the new image.*

### Step 4: Verify Health
Check if the service is up and running.
```bash
curl -f http://localhost:8080/api/v1/health
# Expected: {"status":"up"}
```

### Step 5: Verify Database Connection
Check if the application can connect to the database.
```bash
curl -f http://localhost:8080/api/v1/ready
# Expected: {"status":"up"}
```

### Step 6: Run Smoke Test
Execute the smoke test script against the production URL (use a test account if possible, or verify manually).
```bash
API_URL=https://api.gopickup.com ./scripts/smoke.sh
```

## 3. Rollback Procedure

If the deployment fails or critical bugs are found:

1.  **Revert Image Tag**: Update `docker-compose.prod.yml` to the previous stable version tag.
2.  **Restart Service**:
    ```bash
    docker-compose -f docker-compose.prod.yml up -d server
    ```
3.  **Revert Database (If needed)**:
    - If migrations were destructive or incompatible, you may need to restore from backup (See `docs/BACKUPS.md`).
    - Generally, avoid destructive migrations in production.

## 4. Maintenance Mode (Optional)

If major downtime is expected:
1.  Update Nginx config to return 503 Service Unavailable with a maintenance page.
2.  Stop the application container.
3.  Perform maintenance.
4.  Start application.
5.  Revert Nginx config.
