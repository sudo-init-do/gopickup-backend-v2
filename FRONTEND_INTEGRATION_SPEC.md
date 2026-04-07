# Frontend Integration Specification

**Version**: 1.2.0
**Status**: Updated (Manual WhatsApp Negotiation & Delivery Flow)
**Last Updated**: 2026-04-07

---

## 1. System Overview

The GoPickup backend provides a RESTful API and WebSocket service for a multi-role logistics platform connecting Clients, Drivers, Vendors, and Admins.

- **Base URL (Production)**: `https://backend.gopickup.com.ng/api/v1`
- **WebSocket URL (Production)**: `wss://backend.gopickup.com.ng/api/v1/ws`
- **Authentication**: JWT (Bearer Token)
- **Roles**: `client`, `driver`, `vendor`, `admin`

---

## 2. Order & Delivery Status Flow

The platform follows a **Manual, WhatsApp-mediated flow**.

### 2.1 Order Status Enum
- `pending`: Client placed an order. Negotiation with GoPickup Support has NOT yet started.
- `awaiting_payment`: Negotiation is ongoing (GoPickup Support chat).
- `payment_made`: Client has self-reported that payment was made off-platform.
- `processing`: Admin verified payment. Order is now **visible to all approved drivers** in the app.
- `assigned`: Admin has manually assigned a specific driver (optional manual override).
- `in_progress`: Driver has officially "Accepted" the job in the app.
- `picked_up`: Admin manually marks that the driver has picked up the goods.
- `on_the_way`: Admin manually marks that the goods are in transit.
- `delivered`: Admin manually marks that the delivery is completed.
- `cancelled`: Order was cancelled.

---

## 3. Detailed Frontend Flows

### 3.1 Client Flow (Buying Materials)
1. **Browse**: `GET /products`
2. **Order**: `POST /orders/checkout`
   - **Backend Action**: Order created in `pending` status.
   - **Backend Response**: Returns a `whatsapp_url` that opens a chat with GoPickup Support.
   - **Frontend Action**: Show the "Negotiate on WhatsApp" button using the returned URL.
3. **Negotiation**: Client discusses price, quantity, and payment with Support on WhatsApp.
4. **Report Payment**: After paying the vendor (as agreed on WhatsApp), the client returns to the app and taps "I Have Made Payment."
   - **Endpoint**: `POST /api/v1/orders/:id/payment-made`
   - **Result**: Status moves to `payment_made`.
5. **Monitor**: Status moves to `processing` (after Admin verification), then follows the delivery stages.

### 3.2 Driver Flow (Taking Jobs)
1. **Check Jobs**: `GET /api/v1/jobs/assigned`
   - Shows all jobs in `processing` (available to all) OR jobs specifically `assigned` to this driver.
2. **Negotiate Delivery**: Every job listing includes a `whatsapp_url`.
   - **Frontend Action**: Driver taps "Negotiate on WhatsApp" to discuss the delivery fee with GoPickup Support.
3. **Accept Job**: After agreement on WhatsApp, the driver clicks "I Accept This Job" in the app.
   - **Endpoint**: `POST /api/v1/jobs/:id/accept`
   - **Result**: Order status becomes `in_progress`. The job is now exclusively yours.
4. **Execution**: Driver performs delivery. Status updates (`picked_up`, `on_the_way`, `delivered`) are currently handled by Admin.

### 3.3 Admin Flow (Control Center)
1. **Verify Payment**: After receiving payment confirmation off-platform, Admin verifies the order.
   - **Endpoint**: `POST /api/v1/admin/orders/:id/verify-payment`
   - **Body**: `{ "agreed_price": 50000 }` (optional)
   - **Result**: Status moves to `processing`.
2. **Review Orders**: `GET /api/v1/admin/orders`
3. **Manual Progress Update**: `PATCH /api/v1/admin/orders/status`
   - **Body**: `{ "order_id": "uuid", "status": "picked_up" }`
   - **Options**: `picked_up`, `on_the_way`, `delivered`.

---

## 4. Key Endpoints Reference

### 4.1 Orders (Client)
- `POST /orders/checkout`: Initial checkout (returns `whatsapp_url`).
- `POST /orders/:id/payment-made`: Client signals payment completion.
- `GET /orders`: List my orders.
- `GET /orders/:id`: Details (includes `agreed_price`, `agreed_delivery_fee`).

### 4.2 Jobs (Driver)
- `GET /jobs/assigned`: List all available jobs (processing) and pre-assigned jobs.
- `POST /jobs/:id/accept`: Formally take an available job.

### 4.3 Admin (Restricted)
- `POST /admin/orders/:id/verify-payment`: Confirm payment and release to drivers.
- `PATCH /admin/orders/status`: Manually advance delivery status.
- `POST /admin/orders/assign-driver`: (Optional) Force assign a specific driver.
