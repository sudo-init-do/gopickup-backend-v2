# Smoke Test Checklist

This document provides a comprehensive checklist of `curl` commands to verify the GoPickup backend deployment.
These tests cover all major functionalities: Authentication, Profiles, Marketplace, Orders, Bidding, and Chat.

## Environment Variables
Ensure these are set before running the tests:
```bash
export API_URL="http://localhost:8080/api/v1"
# Or your production URL
# export API_URL="https://api.gopickup.com/api/v1"
```

## 1. Authentication

### 1.1 Register (Client)
```bash
curl -X POST "$API_URL/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "client@test.com",
    "password": "Password123!",
    "role": "client"
  }'
```

### 1.2 Verify OTP (or bypass in dev)
*Note: In production, check email for OTP. In dev, use the seed tool or check logs.*
```bash
# Replace with actual OTP
curl -X POST "$API_URL/auth/verify-otp" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "client@test.com",
    "otp": "123456"
  }'
```

### 1.3 Login (Client) & Save Token
```bash
RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "client@test.com",
    "password": "Password123!"
  }')
CLIENT_TOKEN=$(echo $RESPONSE | jq -r '.token')
echo "Client Token: $CLIENT_TOKEN"
```

### 1.4 Get Me
```bash
curl -X GET "$API_URL/auth/me" \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

### 1.5 Register Vendor & Driver (Repeat steps above)
*Create `vendor@test.com` and `driver@test.com`, verify them, and get their tokens as `VENDOR_TOKEN` and `DRIVER_TOKEN`.*

## 2. Profiles (Admin Approval)

### 2.1 Admin Login
*Ensure admin user exists (use seed tool).*
```bash
ADMIN_RESPONSE=$(curl -s -X POST "$API_URL/auth/login" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "admin@gopickup.com",
    "password": "AdminPassword123!"
  }')
ADMIN_TOKEN=$(echo $ADMIN_RESPONSE | jq -r '.token')
```

### 2.2 Approve Vendor
*Get Vendor ID from database or list pending profiles (if endpoint exists).*
```bash
# Replace UUID with actual Vendor ID
curl -X POST "$API_URL/admin/approve-vendor/VENDOR_UUID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

### 2.3 Approve Driver
```bash
# Replace UUID with actual Driver ID
curl -X POST "$API_URL/admin/approve-driver/DRIVER_UUID" \
  -H "Authorization: Bearer $ADMIN_TOKEN"
```

## 3. Marketplace (Vendor)

### 3.1 Create Product
```bash
curl -X POST "$API_URL/products" \
  -H "Authorization: Bearer $VENDOR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Product",
    "description": "A sample product for smoke testing",
    "price": 100.50,
    "category": "electronics",
    "stock_quantity": 50
  }'
```

### 3.2 List Products (Public)
```bash
curl -X GET "$API_URL/products?page=1&limit=10"
```

## 4. Orders (Client -> Vendor)

### 4.1 Checkout
*Use Product ID from 3.2*
```bash
curl -X POST "$API_URL/orders/checkout" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "items": [
      {"product_id": "PRODUCT_UUID", "quantity": 2}
    ],
    "payment_method": "cash",
    "pickup_address": "123 Vendor St",
    "delivery_address": "456 Client Rd"
  }'
```
*Note Order ID from response.*

### 4.2 Vendor Accept Order
```bash
curl -X POST "$API_URL/orders/ORDER_UUID/accept" \
  -H "Authorization: Bearer $VENDOR_TOKEN"
```

### 4.3 Vendor Mark Ready
```bash
curl -X POST "$API_URL/orders/ORDER_UUID/ready" \
  -H "Authorization: Bearer $VENDOR_TOKEN"
```

## 5. Bidding (Driver -> Client)

### 5.1 Driver Update Location
```bash
curl -X POST "$API_URL/drivers/location" \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "lat": 6.5244,
    "lng": 3.3792
  }'
```

### 5.2 Driver View Available Jobs
```bash
curl -X GET "$API_URL/drivers/jobs" \
  -H "Authorization: Bearer $DRIVER_TOKEN"
```

### 5.3 Driver Bid on Order
```bash
curl -X POST "$API_URL/bids" \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "order_id": "ORDER_UUID",
    "amount": 500.00
  }'
```

### 5.4 Client List Bids
```bash
curl -X GET "$API_URL/orders/ORDER_UUID/bids" \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

### 5.5 Client Accept Bid
```bash
curl -X POST "$API_URL/bids/BID_UUID/accept" \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

## 6. Chat

### 6.1 Initiate Chat (Client with Driver)
```bash
curl -X POST "$API_URL/chats/initiate" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "recipient_id": "DRIVER_UUID",
    "order_id": "ORDER_UUID"
  }'
```

### 6.2 List Chats
```bash
curl -X GET "$API_URL/chats" \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

### 6.3 Get Messages
```bash
curl -X GET "$API_URL/chats/CHAT_UUID/messages" \
  -H "Authorization: Bearer $CLIENT_TOKEN"
```

## 7. Realtime (WebSocket)

### 7.1 Connect
Use a tool like `wscat` or a browser console.
```bash
wscat -c "ws://localhost:8080/ws?token=$CLIENT_TOKEN"
```

### 7.2 Verify Connection
Ensure you receive a welcome message or successful connection status.
