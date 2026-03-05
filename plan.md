# GoPickup Backend Plan (Agent Execution Blueprint)

This file is the single source of truth for how to build the GoPickup backend.
Follow phases in order. Do not skip. Do not invent new endpoints or features.

---

## 0) Global Rules (Non-Negotiable)

### Engineering rules
- Validate all inputs (body, params, query). Reject bad data with clear errors.
- Every protected endpoint requires JWT auth.
- Every role-gated endpoint enforces RBAC (client/driver/vendor/admin).
- Authorization is not “role only”: enforce **resource ownership** (orders/chats/products).
- Use DB transactions for:
  - wallet credit/debit
  - accepting bids (race conditions)
  - checkout stock updates
- All Flutterwave webhook processing is **idempotent by reference** (never double credit).
- Log important state changes: order status changes, wallet transactions, approvals.

### Output rules (for the Agent)
After each phase, the agent must output:
1. What was implemented (bullet list)
2. How to run it locally (commands)
3. What tests were added (list)
4. Proof that “Done When” is met (explicit)

---

## 1) Tech Decisions (Lock These In Before Coding)

### Recommended Stack (Pick One and stick to it)
- **Language**: Go
- **HTTP**: Gin (or Fiber)
- **DB**: PostgreSQL + PostGIS
- **Realtime**: WebSockets (Gorilla/websocket) or Socket.IO equivalent if using Node
- **Cache / jobs (optional)**: Redis
- **Storage**: S3 or Cloudinary
- **Payments**: Flutterwave
- **Push**: FCM

> If agent is already using Go + Gin + GORM, continue with that.

---

## 2) Phase 0: Repo Setup + Boot

### Goal
Project runs locally, has Docker for DB, and exposes /health.

### Tasks
- [ ] Create folders:
  - cmd/server
  - internal/config
  - internal/db
  - internal/models
  - internal/repositories
  - internal/services
  - internal/http (routes/handlers)
  - internal/middleware
  - internal/realtime
  - internal/utils
- [ ] Add `.env.example`
- [ ] Add `docker-compose.yml`:
  - Postgres with PostGIS enabled
  - API service (optional) or run API locally
- [ ] Add `GET /health -> 200 {status:"ok"}`

### Done When
- `docker compose up -d` starts DB
- API starts and `GET /health` returns 200

---

## 3) Phase 1: Database Schema + Migrations

### Goal
All tables exist with correct constraints and indexes. PostGIS enabled.

### MUST-HAVE tables
#### users
- id UUID PK
- email UNIQUE
- password_hash
- role ENUM(client, driver, vendor, admin)
- is_verified BOOLEAN default false
- fcm_token nullable
- created_at, updated_at

#### profiles (1:1 with users)
client_profiles:
- user_id UUID PK FK(users.id)
- full_name
- phone_number UNIQUE
- address
- profile_picture_url nullable

driver_profiles:
- user_id UUID PK FK(users.id)
- full_name
- phone_number UNIQUE
- license_number UNIQUE
- vehicle_type ENUM(Tricycle, Van, Truck, Flatbed, Trailer)
- plate_number UNIQUE
- vehicle_capacity
- is_approved BOOLEAN default false
- current_location_lat nullable
- current_location_lng nullable
- location GEOGRAPHY(Point,4326) nullable (PostGIS source-of-truth)
- profile_picture_url nullable

vendor_profiles:
- user_id UUID PK FK(users.id)
- store_name
- business_type
- address
- store_banner_url nullable
- is_approved BOOLEAN default false

#### wallets
- user_id UUID PK FK(users.id)
- balance DECIMAL default 0.00
- currency default NGN

#### transactions
- id UUID PK
- user_id FK
- amount DECIMAL
- type ENUM(deposit, withdrawal, payment, payout)
- reference UNIQUE
- status ENUM(pending, successful, failed)
- created_at

#### products
- id UUID PK
- vendor_id UUID FK(users.id)
- name
- description
- price DECIMAL
- category
- stock_quantity INT
- image_url
- is_active BOOLEAN default true
- created_at, updated_at

#### orders
- id UUID PK
- client_id FK(users.id)
- vendor_id FK(users.id)
- driver_id FK(users.id) nullable
- status ENUM(pending, processing, searching_driver, transit, delivered, cancelled)
- total_product_amount DECIMAL
- delivery_fee DECIMAL nullable
- pickup_address
- delivery_address
- delivery_lat FLOAT
- delivery_lng FLOAT
- payment_method ENUM(wallet, card, cash_on_delivery)
- created_at, updated_at

#### order_items
- id UUID PK
- order_id FK
- product_id FK
- quantity INT
- price_at_time_of_purchase DECIMAL

#### job_bids
- id UUID PK
- order_id FK
- driver_id FK
- amount DECIMAL
- estimated_time STRING
- status ENUM(pending, accepted, rejected)
- created_at
- UNIQUE(order_id, driver_id)

#### conversations
- id UUID PK
- participant_1_id FK(users.id)
- participant_2_id FK(users.id)
- order_id FK(orders.id) nullable
- created_at

#### messages
- id UUID PK
- conversation_id FK
- sender_id FK
- text TEXT
- is_read BOOLEAN default false
- created_at

### Indexes (minimum)
- orders(status), orders(client_id), orders(vendor_id), orders(driver_id)
- products(vendor_id), products(category), products(is_active)
- job_bids(order_id), job_bids(driver_id)
- messages(conversation_id, created_at)

### Done When
- migrations run cleanly
- constraints prevent duplicates
- PostGIS extension enabled

---

## 4) Phase 2: Auth + OTP + RBAC

### Goal
Register/login + JWT + RBAC. OTP verification flips is_verified.

### Endpoints
- [ ] POST /api/v1/auth/register
- [ ] POST /api/v1/auth/verify-otp
- [ ] POST /api/v1/auth/login
- [ ] POST /api/v1/auth/forgot-password
- [ ] POST /api/v1/auth/reset-password
- [ ] GET  /api/v1/auth/me

### Required middleware
- [ ] AuthMiddleware (JWT)
- [ ] RoleGuard(roles...)

### Done When
- protected routes require JWT
- wrong roles are blocked
- verified flag enforced as designed

---

## 5) Phase 3: Profiles + Admin Approval [DONE]

### Goal
Onboarding works for each role. Admin can approve drivers/vendors.

### Endpoints
- [x] POST /api/v1/profile/client (Client)
- [x] POST /api/v1/profile/driver (Driver)
- [x] POST /api/v1/profile/vendor (Vendor)
- [x] PUT  /api/v1/profile (Any)

### Add Admin endpoints (required even if not listed)
- [x] PATCH /api/v1/admin/drivers/:user_id/approve
- [x] PATCH /api/v1/admin/vendors/:user_id/approve

### Rules
- driver/vendor cannot bid/sell until approved

### Done When
- [x] onboarding saves data correctly
- [x] approval gates are enforced

---

## 6) Phase 4: Marketplace [COMPLETED]

### Goal
Vendor product CRUD + public browsing.

### Vendor endpoints
- [x] POST   /api/v1/vendor/products
- [x] PUT    /api/v1/vendor/products/:id
- [x] DELETE /api/v1/vendor/products/:id (soft delete preferred)
- [x] GET    /api/v1/vendor/dashboard

### Public endpoints
- [x] GET /api/v1/products (filters + pagination)
- [x] GET /api/v1/products/:id
- [x] GET /api/v1/vendors (filters + pagination)

### Done When
- [x] vendor CRUD works + ownership enforced
- [x] public list paginates and filters

---

## 7) Phase 5: Orders Lifecycle (State Machine) [COMPLETED]

### Goal
Checkout works + order status transitions enforced.

### Endpoints
- [x] POST  /api/v1/orders/checkout (Client)
- [x] GET   /api/v1/orders (role-aware)
- [x] GET   /api/v1/orders/:id (authorized actors only)
- [x] PATCH /api/v1/orders/:id/status (Vendor: pending->processing or cancel)
- [x] PATCH /api/v1/orders/:id/ready (Vendor: processing->searching_driver)

### State rules
- pending -> processing OR cancelled
- processing -> searching_driver OR cancelled
- searching_driver -> transit ONLY via bid acceptance
- transit -> delivered (Driver/admin controlled later)
- cancelled is terminal

### Done When
- [x] invalid transitions rejected
- [x] unauthorized users cannot view/modify orders

---

## 8) Phase 6: Driver Jobs + Bidding [COMPLETED]

### Goal
Driver sees nearby jobs (PostGIS), bids, client accepts safely.

### Endpoints
- [x] PATCH /api/v1/driver/location (Driver updates location)
- [x] GET   /api/v1/jobs/available (Driver; PostGIS radius)
- [x] POST  /api/v1/jobs/:order_id/bid (Driver)
- [x] GET   /api/v1/orders/:id/bids (Client)
- [x] POST  /api/v1/orders/:id/bids/:bid_id/accept (Client)

### Transaction requirements
Bid acceptance MUST be a DB transaction:
- accept chosen bid
- reject others
- set order.driver_id
- set status transit

### Done When
- [x] only one driver can be assigned even under race conditions
- [x] new bids notify client (later via WS/FCM)

---

## 9) Phase 7: Wallet + Flutterwave (Idempotent)

### Goal
Funding, webhook, withdrawal.

### Endpoints
- [ ] GET  /api/v1/wallet/balance
- [ ] POST /api/v1/wallet/fund (create pending tx + payment link)
- [ ] POST /api/v1/wallet/webhook (verify hash + idempotent by reference)
- [ ] POST /api/v1/wallet/withdraw (vendor/driver)

### Done When
- webhook replay does not double-credit
- wallet balance correct

---

## 10) Phase 8: Realtime (WebSockets) + Push (FCM) [COMPLETED]

### Goal
Rooms + events exactly as spec. FCM alongside key events.

### Rooms
- user:{user_id}
- order:{order_id}
- chat:{chat_id}

### Client -> Server events
- [x] driver_location_update {lat,lng,order_id} every 10s in transit
- [x] chat_message {chat_id,text}

### Server -> Client events
- [x] order_status_updated
- [x] new_bid
- [x] bid_accepted
- [x] driver_moved
- [x] new_message

### Done When
- [x] events broadcast correctly
- [x] FCM also sent for bid/status/message when app closed

---

## 11) Phase 9: Tests (Minimum)

### Required tests
- [ ] Auth flow (register->verify->login->me)
- [ ] RBAC blocks wrong roles
- [ ] Order transition rules enforced
- [ ] Bid accept race safety (one assignment)
- [ ] Webhook idempotency (no double credit)

### Done When
- tests pass in CI or local run

---

# How the Agent Should Work
- Implement one phase at a time.
- Do not start next phase until current phase “Done When” is satisfied.
- Keep API base path: /api/v1
for mailing we are using plunk 
- Keep responses consistent (errors, pagination).