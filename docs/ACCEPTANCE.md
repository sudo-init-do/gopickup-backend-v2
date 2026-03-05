# Acceptance Checklist

This document serves as the final sign-off checklist for the GoPickup Backend (Phase 12).
Ensure each item is verified in the Staging environment.

## 1. Authentication & Onboarding
- [ ] **Admin Account**: Can be seeded via `ALLOW_SEED=true` or CLI tool.
- [ ] **Client Registration**: New client can register and receives OTP email (mocked or real).
- [ ] **Client Login**: Can login with correct credentials -> Receives JWT.
- [ ] **Vendor Registration**: Vendor can register, profile is `pending`.
- [ ] **Driver Registration**: Driver can register, profile is `pending`.
- [ ] **Admin Approval**: Admin can approve pending Vendor/Driver profiles via API/DB.

## 2. Product Management (Vendor)
- [ ] **Create Product**: Vendor can create a product.
- [ ] **List Products**: Public endpoint `/api/v1/products` lists active products.
- [ ] **Update/Delete**: Vendor can modify their own products.
- [ ] **Stock Management**: Inventory decreases upon checkout (or order confirmation).

## 3. Order Lifecycle
- [ ] **Checkout**: Client can create an order with valid items.
- [ ] **Order Created**: Order status is `PENDING`.
- [ ] **Vendor Notification**: Vendor sees new order in dashboard.
- [ ] **Vendor Action**: Vendor can mark order as `READY`. Status updates to `READY`.

## 4. Driver & Bidding
- [ ] **Location Update**: Driver can update lat/long.
- [ ] **Job Discovery**: Driver sees available `READY` orders within radius.
- [ ] **Place Bid**: Driver can place a bid on a job.
- [ ] **Client Notification**: Client receives notification of new bid.
- [ ] **View Bids**: Client can list all bids for an order.
- [ ] **Accept Bid**: Client accepts a bid.
    - [ ] Order status becomes `ASSIGNED` (or `TRANSIT`).
    - [ ] Other bids are rejected/closed.
    - [ ] Driver is notified.

## 5. Realtime Communication (Chat & Updates)
- [ ] **WebSocket Connection**: Client/Driver can connect to `wss://.../ws` with valid JWT.
- [ ] **Initiate Chat**: Client/Driver can start a chat for an active order.
- [ ] **Send Message**: Message sent via REST API appears on WebSocket for recipient.
- [ ] **History**: Fetching chat history returns previous messages.

## 6. Security & Infrastructure
- [ ] **Health Check**: `GET /api/v1/health` returns 200 OK.
- [ ] **Database Connectivity**: `GET /api/v1/ready` returns 200 OK.
- [ ] **CORS**: Requests from allowed origins succeed; others fail.
- [ ] **Rate Limiting**: Spamming auth endpoints triggers 429 Too Many Requests.
- [ ] **Environment**: `APP_ENV=staging` is set correctly.

## 7. Documentation
- [ ] **API Docs**: Swagger/Postman collection is available and matches implementation.
- [ ] **Setup Guide**: `STAGING_SETUP.md` is clear and accurate.
- [ ] **Smoke Test**: `scripts/smoke.sh` runs successfully against staging.
