# Frontend Integration Specification

**Version**: 1.1.0
**Status**: Updated (Manual Order & Delivery Flow)
**Last Updated**: 2026-03-26

---

## 1. System Overview

The GoPickup backend provides a RESTful API and WebSocket service for a multi-role logistics platform connecting Clients, Drivers, Vendors, and Admins.

- **Base URL (Production)**: `https://backend.gopickup.com.ng/api/v1`
- **WebSocket URL (Production)**: `wss://backend.gopickup.com.ng/api/v1/ws`
- **Authentication**: JWT (Bearer Token)
- **Roles**: `client`, `driver`, `vendor`, `admin`

---

## 2. Order & Delivery Status Flow

The platform has moved from an automated bidding system to a **Manual, WhatsApp-mediated, Admin-controlled flow**.

### 2.1 Order Status Enum
- `pending`: Client placed an order for vendor materials.
- `processing`: Negotiation/Payment status (User should be directed to WhatsApp Support).
- `assigned`: Admin has assigned a specific driver to this load.
- `in_progress`: Driver has officially "Accepted" the load in the app after WhatsApp negotiation.
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
   - **Frontend Action**: Redirect user to WhatsApp Support Link for price negotiation and payment.
3. **Monitor**: Listen for `order_status_updated` via WebSocket.
   - Status moves to `assigned` once Admin picks a driver.
   - Status moves to `in_progress` once the Driver accepts.
   - Status moves to `picked_up`, `on_the_way`, and `delivered` as Admin updates progress.

### 3.2 Driver Flow (Taking Loads)
1. **Check Assigned Loads**: `GET /api/v1/jobs/assigned`
   - Only shows loads specifically assigned to this driver with status `assigned`.
2. **Negotiate**: The driver sees a "Chat with Support" (WhatsApp) button on the load.
3. **Accept**: After WhatsApp agreement, the driver clicks "Accept Task" in the app.
   - **Endpoint**: `POST /api/v1/jobs/:id/accept`
   - **Result**: Order status becomes `in_progress`.
4. **Execution**: The driver simply delivers the goods. All status updates to the client (picked up, on the way, etc.) are handled manually by the Admin, not the Driver.

### 3.3 Admin Flow (Control Center)
1. **View Orders**: `GET /api/v1/admin/orders`
2. **Assign Driver**:
   - **Endpoint**: `POST /admin/orders/assign-driver`
   - **Body**:
     ```json
     {
       "order_id": "uuid",
       "driver_id": "uuid",
       "agreed_price": 50000,
       "delivery_fee": 5000
     }
     ```
   - **Result**: Order status becomes `assigned`. Driver is notified.
3. **Manual Progress Update**:
   - **Endpoint**: `PATCH /admin/orders/status`
   - **Body**: `{ "order_id": "uuid", "status": "picked_up" }`
   - **Options**: `picked_up`, `on_the_way`, `delivered`.

---

## 4. Key Endpoints Reference

### 4.1 Orders (Client/Shared)
- `POST /orders/checkout`: Place initial order.
- `PATCH /orders/:id/cancel`: Cancel a pending order (Client only).
- `GET /orders`: List my orders.
- `GET /orders/:id`: Get order details (includes `agreed_price` and `agreed_delivery_fee`).

### 4.2 Jobs (Driver)
- `GET /jobs/assigned`: List my specifically assigned loads.
- `POST /jobs/:id/accept`: Formally accept an assigned load.

### 4.3 Admin (Restricted)
- `POST /admin/orders/assign-driver`: Assign a driver and set final agreed prices.
- `PATCH /admin/orders/status`: Manually advance the delivery status for the client.
- `GET /admin/users`: List users to find a driver ID for assignment.

---

## 5. QA Checklist for New Flow
- [ ] Client: Checkout product (Status = `pending`).
- [ ] Admin: Assign a driver (Status = `assigned`).
- [ ] Driver: See job in `assigned` list (NOT in a public public board).
- [ ] Driver: Accept job (Status = `in_progress`).
- [ ] Admin: Mark as `picked_up` (Client sees update).
- [ ] Admin: Mark as `on_the_way` (Client sees update).
- [ ] Admin: Mark as `delivered` (Order completes).
