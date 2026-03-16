# Frontend Integration Specification

**Version**: 1.0.0
**Status**: Draft
**Last Updated**: 2026-03-06

---

## 1. System Overview

The GoPickup backend provides a RESTful API and WebSocket service for a multi-role logistics platform connecting Clients, Drivers, and Vendors.

- **Base URL (Production)**: `https://backend.gopickup.com.ng/api/v1`
- **WebSocket URL (Production)**: `wss://backend.gopickup.com.ng/api/v1/ws`
- **Base URL (Local)**: `http://localhost:8080/api/v1`
- **WebSocket URL (Local)**: `ws://localhost:8080/api/v1/ws`
- **Authentication**: JWT (Bearer Token)
- **Roles**: `client`,    `driver`, `vendor`, `admin`

### 1.1 Service Health Check
The backend provides a public health check endpoint to verify service availability.
- **Endpoint**: `GET /` or `GET /health` or `GET /api/v1/health`
- **Response**: `200 OK`
  ```json
  {
    "status": "ok"
  }
  ```

---

## 2. Global API Rules

### 2.1 Authentication
All protected endpoints require the `Authorization` header:
```
Authorization: Bearer <your_jwt_token>
```

### 2.2 Response Format
**Success Response**:
```json
{
  "id": "uuid",
  "created_at": "timestamp",
  ...resource_fields
}
```

**Paginated Response**:
```json
{
  "data": [ ...items ],
  "meta": {
    "current_page": 1,
    "total_pages": 5,
    "total_items": 42,
    "limit": 10
  }
}
```

**Error Response**:
```json
{
  "error": "Description of the error"
}

**Common Status Codes**:
- `200 OK`: Success
- `201 Created`: Resource created
- `400 Bad Request`: Validation error
- `401 Unauthorized`: Missing or invalid token
- `403 Forbidden`: Insufficient role permissions
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server failure

### 2.3 Date & Time
All timestamps are returned in **ISO 8601** format (e.g., `2023-10-27T10:00:00Z`).

---

## 3. Data Models

### 3.1 User & Profile
**User Role Enum**: `client`, `driver`, `vendor`, `admin`

**Client Profile**:
- `full_name` (string)
- `phone_number` (string)
- `address` (string)
- `profile_picture_url` (string, optional)

**Driver Profile**:
- `full_name` (string)
- `phone_number` (string)
- `license_number` (string)
- `vehicle_type` (enum: `Tricycle`, `Van`, `Truck`, `Flatbed`, `Trailer`)
- `plate_number` (string)
- `vehicle_capacity` (float)
- `is_approved` (bool)
- `profile_picture_url` (string, optional)

**Vendor Profile**:
- `store_name` (string)
- `business_type` (string)
- `address` (string)
- `phone_number` (string)
- `store_banner_url` (string, optional)
- `is_approved` (bool)

### 3.2 Product
- `id` (uuid)
- `vendor_id` (uuid)
- `name` (string)
- `description` (string)
- `price` (float)
- `stock_quantity` (int)
- `category` (string)
- `image_url` (string)
- `is_active` (bool)

### 3.3 Order
**Status Enum**: `pending`, `processing`, `searching_driver`, `assigned`, `picked_up`, `delivered`, `cancelled`
**Payment Method Enum**: `wallet`, `card`, `cash_on_delivery`

- `id` (uuid)
- `client_id` (uuid)
- `vendor_id` (uuid)
- `driver_id` (uuid, optional)
- `items` (array of OrderItem)
- `total_product_amount` (float)
- `status` (OrderStatus)
- `pickup_address` (string)
- `delivery_address` (string)
- `delivery_lat` (float, optional)
- `delivery_lng` (float, optional)
- `created_at` (timestamp)

### 3.4 Bid
**Status Enum**: `pending`, `accepted`, `rejected`

- `id` (uuid)
- `order_id` (uuid)
- `driver_id` (uuid)
- `amount` (float)
- `status` (BidStatus)

---

## 4. Authentication APIs

### 4.1 Register
`POST /auth/register`
**Request**:
```json
{
  "email": "user@example.com",
  "password": "password123",
  "role": "client" // or driver, vendor
}
```
**Response**: `{ "token": "jwt...", "user": { ... } }`

### 4.2 Login
`POST /auth/login`
**Request**:
```json
{
  "email": "user@example.com",
  "password": "password123"
}
```
**Response**: `{ "token": "jwt...", "user": { ... } }`

### 4.3 Get Current User
`GET /auth/me`
**Response**: User object with profile data.

---

## 5. Profile APIs

### 5.1 Create Profile
- Client: `POST /profile/client` (Body: Client Profile fields)
- Driver: `POST /profile/driver` (Body: Driver Profile fields)
- Vendor: `POST /profile/vendor` (Body: Vendor Profile fields)

### 5.2 Update Profile
`PUT /profile`
**Request**: JSON object with fields to update (partial updates supported).

---

## 6. Marketplace APIs (Public & Client)

### 6.1 List Products
`GET /products`
**Query Params**:
- `category`: Filter by category
- `min_price`, `max_price`: Price range
- `vendor_id`: Filter by vendor
- `search`: Search by name/description
- `page`, `limit`: Pagination

### 6.2 Get Product Details
`GET /products/:id`

### 6.3 List Vendors
`GET /vendors`
**Query Params**: `business_type`, `search`, `page`, `limit`

---

## 7. Order APIs

### 7.1 Checkout (Client)
`POST /orders/checkout`
**Request**:
```json
{
  "items": [
    { "product_id": "uuid", "quantity": 2 }
  ],
  "payment_method": "cash_on_delivery",
  "pickup_address": "123 Store St",
  "delivery_address": "456 Home Ave",
  "delivery_lat": 12.34, // Optional
  "delivery_lng": 56.78  // Optional
}
```
**Note**: All items must belong to the same Vendor.

### 7.2 List Orders
`GET /orders`
**Query Params**: `page`, `limit`
**Response**: List of orders relevant to the user's role.

### 7.3 Get Order Details
`GET /orders/:id`

### 7.4 Get Bids (Client)
`GET /orders/:id/bids`
**Response**: List of bids for the order.

### 7.5 Accept Bid (Client)
`POST /orders/:id/bids/:bid_id/accept`
**Response**: Updated Order object (status -> `assigned`).

---

## 8. Vendor APIs

### 8.1 Manage Products
- `POST /vendor/products`: Create product
- `PUT /vendor/products/:id`: Update product
- `DELETE /vendor/products/:id`: Delete product

### 8.2 Order Management
- `PATCH /vendor/orders/:id/status`: Update status (Body: `{ "status": "processing" }`)
- `PATCH /vendor/orders/:id/ready`: Mark order as ready for pickup.

### 8.3 Dashboard
`GET /vendor/dashboard`
**Response**: `{ "total_sales": 100.0, "active_orders": 5 }`

---

## 9. Driver APIs

### 9.1 Update Location
`PATCH /driver/location`
**Request**: `{ "lat": 12.34, "lng": 56.78 }`

### 9.2 Get Available Jobs
`GET /jobs/available`
**Response**: List of orders with status `searching_driver`.

### 9.3 Place Bid
`POST /jobs/:order_id/bid`
**Request**: `{ "amount": 150.00 }`

---

## 10. Chat APIs

### 10.1 Initiate Chat
`POST /chats/initiate`
**Request**:
```json
{
  "recipient_user_id": "uuid",
  "order_id": "uuid" // Optional
}
```

### 10.2 List Chats
`GET /chats`
**Response**: List of chat conversations with `last_message` and `unread_count`.

### 10.3 Get Messages
`GET /chats/:id/messages`
**Query Params**: `page`, `limit`

### 10.4 Mark Read
`PATCH /chats/:id/read`

---

## 11. WebSocket Integration

### 11.1 Connection
Connect to: `ws://localhost:8080/api/v1/ws?token=<JWT_TOKEN>`

### 11.2 Client -> Server Events
Send JSON payloads to the WebSocket connection.

**Join Order Room** (To receive real-time updates for an order):
```json
{
  "event": "join_order_room",
  "payload": { "order_id": "uuid" }
}
```

**Leave Order Room**:
```json
{
  "event": "leave_order_room",
  "payload": { "order_id": "uuid" }
}
```

**Join Chat Room**:
```json
{
  "event": "join_chat_room",
  "payload": { "chat_id": "uuid" }
}
```

**Send Chat Message**:
```json
{
  "event": "chat_message",
  "payload": {
    "chat_id": "uuid",
    "text": "Hello, I'm here!"
  }
}
```

**Driver Location Update** (Driver only, frequent updates):
```json
{
  "event": "driver_location_update",
  "payload": {
    "order_id": "uuid",
    "lat": 12.34,
    "lng": 56.78
  }
}
```

### 11.3 Server -> Client Events
Listen for these JSON messages from the server.

**Order Status Updated**:
```json
{
  "event": "order_status_updated",
  "payload": {
    "order_id": "uuid",
    "status": "picked_up"
  }
}
```

**New Bid Received** (Client only):
```json
{
  "event": "new_bid",
  "payload": {
    "order_id": "uuid",
    "amount": 150.00
  }
}
```

**Bid Accepted** (Driver only):
```json
{
  "event": "bid_accepted",
  "payload": { "order_id": "uuid" }
}
```

**Driver Moved** (For tracking):
```json
{
  "event": "driver_moved",
  "payload": {
    "order_id": "uuid",
    "lat": 12.3456,
    "lng": 56.7890,
    "ts": "2023-..."
  }
}
```

**New Chat Message**:
```json
{
  "event": "new_message",
  "payload": {
    "chat_id": "uuid",
    "sender_id": "uuid",
    "content": "Hello!",
    "ts": "2023-..."
  }
}
```

**General Notification** (Push Notification fallback):
```json
{
  "event": "notification",
  "payload": {
    "title": "Order Update",
    "body": "Your order has been delivered",
    "data": { "type": "order_status", "id": "uuid" }
  }
}
```

---

## 12. File Upload APIs

### 12.1 Upload Image
`POST /upload`
**Content-Type**: `multipart/form-data`
**Request**:
- `image`: The image file to upload (Field name: `image`, supported extensions: `.jpg`, `.jpeg`, `.png`, `.webp`)

**Response**:
```json
{
  "message": "File uploaded successfully",
  "image_url": "/uploads/123e4567-e89b-12d3-a456-426614174000.png"
}
```

---

## 13. Feature Flags & Configuration
- **Max Image Size**: 10MB
- **Rate Limits**:
  - Auth: 5 requests / 10s
  - Driver Location: 1 update / 5s
  - Chat: 1 message / 1s

## 14. Frontend Flows

### 14.1 Order Lifecycle (Client View)
1.  **Browse**: `GET /products`
2.  **Checkout**: `POST /orders/checkout` -> Order `pending`
3.  **Wait for Vendor**: Listen for `order_status_updated` (`processing` -> `searching_driver`)
4.  **Wait for Bids**: Listen for `new_bid`.
5.  **Accept Bid**: `POST /orders/:id/bids/:bid_id/accept` -> Order `assigned`.
6.  **Track Driver**: Listen for `driver_moved`.
7.  **Completion**: Listen for `order_status_updated` (`delivered`).

### 13.2 Driver Job Flow
1.  **Find Jobs**: `GET /jobs/available`
2.  **Bid**: `POST /jobs/:order_id/bid`
3.  **Wait**: Listen for `bid_accepted`.
4.  **Execute**:
    -   Update location periodically via WS `driver_location_update`.
    -   (Backend handles status updates based on location/actions, or Admin/Vendor triggers for now).

---

## 15. QA Smoke Test Checklist
- [ ] Register new Client account
- [ ] Register new Driver account & approve via Admin API
- [ ] Register new Vendor account & create 1 product
- [ ] Client: Checkout product (Order Created)
- [ ] Vendor: Update status to `searching_driver`
- [ ] Driver: See job in `available` list & Place Bid
- [ ] Client: See bid & Accept it
- [ ] Driver: Receive `bid_accepted` event
- [ ] Chat: Client initiates chat with Driver
- [ ] WebSocket: Verify real-time message delivery
