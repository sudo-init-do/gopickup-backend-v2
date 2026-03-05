#!/bin/bash
set -e

# Configuration
API_URL=${API_URL:-"http://localhost:8080/api/v1"}
ADMIN_EMAIL="admin@test.com"
VENDOR_EMAIL="vendor@test.com"
DRIVER_EMAIL="driver@test.com"
CLIENT_EMAIL="client@test.com"
PASSWORD="Password123!"

echo "Starting Smoke Test against $API_URL"
echo "Ensure you have seeded the database first!"

# Helper to check for errors
check_error() {
  if echo "$1" | grep -q "error"; then
    echo "Request failed: $1"
    exit 1
  fi
}

# 1. Login
echo "---------------------------------------------------"
echo "Logging in..."

echo "-> Vendor Login"
RESP=$(curl -s -X POST "$API_URL/auth/login" -H "Content-Type: application/json" -d "{\"email\":\"$VENDOR_EMAIL\", \"password\":\"$PASSWORD\"}")
check_error "$RESP"
VENDOR_TOKEN=$(echo $RESP | jq -r '.token')

echo "-> Client Login"
RESP=$(curl -s -X POST "$API_URL/auth/login" -H "Content-Type: application/json" -d "{\"email\":\"$CLIENT_EMAIL\", \"password\":\"$PASSWORD\"}")
check_error "$RESP"
CLIENT_TOKEN=$(echo $RESP | jq -r '.token')

echo "-> Driver Login"
RESP=$(curl -s -X POST "$API_URL/auth/login" -H "Content-Type: application/json" -d "{\"email\":\"$DRIVER_EMAIL\", \"password\":\"$PASSWORD\"}")
check_error "$RESP"
DRIVER_TOKEN=$(echo $RESP | jq -r '.token')

echo "-> Tokens acquired."

# 2. Vendor Create Product
echo "---------------------------------------------------"
echo "Creating Product..."
RESP=$(curl -s -X POST "$API_URL/vendor/products" \
  -H "Authorization: Bearer $VENDOR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name":"Smoke Test Product", "description":"A test product", "price": 100, "stock_quantity": 100, "is_active": true}')
check_error "$RESP"
PRODUCT_ID=$(echo $RESP | jq -r '.id')
echo "-> Product Created: $PRODUCT_ID"

# 3. Client Checkout
echo "---------------------------------------------------"
echo "Client Checkout..."
RESP=$(curl -s -X POST "$API_URL/orders/checkout" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"items\":[{\"product_id\":\"$PRODUCT_ID\", \"quantity\":1}], \"payment_method\":\"cash\", \"pickup_address\":\"Vendor Loc\", \"delivery_address\":\"Client Loc\", \"delivery_lat\": 10.0, \"delivery_lng\": 20.0}")
check_error "$RESP"
ORDER_ID=$(echo $RESP | jq -r '.id')
echo "-> Order Created: $ORDER_ID"

# 4. Vendor Mark Ready
echo "---------------------------------------------------"
echo "Vendor Marking Order Ready..."
RESP=$(curl -s -X PATCH "$API_URL/vendor/orders/$ORDER_ID/ready" \
  -H "Authorization: Bearer $VENDOR_TOKEN")
check_error "$RESP"
echo "-> Order marked as ready."

# 5. Driver Update Location & Check Jobs
echo "---------------------------------------------------"
echo "Driver Updating Location..."
RESP=$(curl -s -X PATCH "$API_URL/driver/location" \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"lat": 10.01, "lng": 20.01}')
check_error "$RESP"

echo "Driver Checking Available Jobs..."
RESP=$(curl -s -X GET "$API_URL/jobs/available?lat=10.01&lng=20.01&radius=10" \
  -H "Authorization: Bearer $DRIVER_TOKEN")
check_error "$RESP"
# Just verify we get a list (starts with [)
if [[ $RESP != [* ]]; then
    echo "Warning: Unexpected response format for jobs: $RESP"
fi
echo "-> Jobs checked."

# 6. Driver Bid
echo "---------------------------------------------------"
echo "Driver Placing Bid..."
RESP=$(curl -s -X POST "$API_URL/jobs/$ORDER_ID/bid" \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 150.0}')
check_error "$RESP"
BID_ID=$(echo $RESP | jq -r '.id')
echo "-> Bid Placed: $BID_ID"

# 7. Client Accept Bid
echo "---------------------------------------------------"
echo "Client Accepting Bid..."
RESP=$(curl -s -X POST "$API_URL/orders/$ORDER_ID/bids/$BID_ID/accept" \
  -H "Authorization: Bearer $CLIENT_TOKEN")
check_error "$RESP"
echo "-> Bid Accepted. Order should be in Transit/Assigned."

# 8. Chat Initiation
echo "---------------------------------------------------"
echo "Client Initiating Chat with Driver..."
# Need Driver User ID. We can get it from the Bid or Order details usually, but here we know the seeded Driver ID is harder to guess if we didn't save it.
# However, fetching the order details as Client should reveal the assigned DriverID.
RESP=$(curl -s -X GET "$API_URL/orders/$ORDER_ID" \
  -H "Authorization: Bearer $CLIENT_TOKEN")
check_error "$RESP"
ASSIGNED_DRIVER_ID=$(echo $RESP | jq -r '.driver_id')

if [ "$ASSIGNED_DRIVER_ID" == "null" ]; then
  echo "Error: Order does not have an assigned driver."
  exit 1
fi

RESP=$(curl -s -X POST "$API_URL/chats/initiate" \
  -H "Authorization: Bearer $CLIENT_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"recipient_id\":\"$ASSIGNED_DRIVER_ID\", \"order_id\":\"$ORDER_ID\"}")
check_error "$RESP"
CHAT_ID=$(echo $RESP | jq -r '.id')
echo "-> Chat Initiated: $CHAT_ID"

echo "---------------------------------------------------"
echo "Smoke Test Completed Successfully!"
